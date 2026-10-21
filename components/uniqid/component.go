package uniqid

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/uniqid/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Component struct {
	node    *snowflake.Node
	config  *config.Config
	etcdCli *clientv3.Client

	wg        *sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	serverId  int
	serverIds *ServerIds
	ch        chan *Event
	mainCh    chan *Event
	chs       []chan *Event

	mu         sync.RWMutex
	started    bool
	closing    bool
	leaseUntil time.Time
	leaseValid bool
	closeErr   error
	done       chan struct{}
}

func New(cfg *config.Config) *Component {
	me := &Component{
		config: cfg,
		done:   make(chan struct{}),
		wg:     new(sync.WaitGroup),
		ch:     make(chan *Event, 1),
		mainCh: make(chan *Event, 1),
		serverIds: &ServerIds{
			Data: make(map[int]bool),
		},
	}
	me.ctx, me.cancel = context.WithCancel(context.Background())
	return me
}

func (me *Component) Start() error {
	me.mu.Lock()
	if me.started || me.closing {
		me.mu.Unlock()
		return errors.New("uniqid already started or closed")
	}
	if me.etcdCli == nil {
		me.mu.Unlock()
		return errors.New("etcd client is nil")
	}
	me.started = true
	me.wg.Add(2)
	go me.dispatchEvents()
	go me.mainloop()
	go func() { me.wg.Wait(); close(me.done) }()
	me.mu.Unlock()
	timer := time.NewTimer(me.config.InitTimeout)
	defer timer.Stop()
	for {
		select {
		case evt := <-me.mainCh:
			if evt.Type == TypeServerIdUpdate {
				me.mu.RLock()
				ready := me.leaseValid && !me.closing && time.Now().Before(me.leaseUntil)
				me.mu.RUnlock()
				if ready {
					return nil
				}
			}
			if evt.Type == TypeError {
				return stderrors.Join(evt.Error, me.Close())
			}
		case <-me.ctx.Done():
			return stderrors.Join(me.ctx.Err(), me.Close())
		case <-timer.C:
			return stderrors.Join(errors.New("New: timeout"), me.Close())
		}
	}
}

func (me *Component) Close() error {
	me.mu.Lock()
	me.closing = true
	me.leaseValid = false
	started := me.started
	me.cancel()
	me.mu.Unlock()
	if !started {
		return nil
	}
	select {
	case <-me.done:
	default:
		timer := time.NewTimer(me.config.CloseTimeout)
		defer timer.Stop()
		select {
		case <-me.done:
		case <-timer.C:
			return errors.WithMessage(context.DeadlineExceeded, "Close")
		}
	}
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.closeErr
}

// Generate 仅在有效租约内发号；失去租约、未启动或已关闭时按公开契约 panic。
func (me *Component) Generate() snowflake.ID {
	me.mu.RLock()
	defer me.mu.RUnlock()
	if me.closing || !me.leaseValid || me.node == nil || !time.Now().Before(me.leaseUntil) {
		panic("uniqid: no valid lease")
	}
	id := me.node.Generate()
	// 雪花序列耗尽时可能等待下一毫秒；返回前再次确认租约没有过期。
	if !time.Now().Before(me.leaseUntil) {
		panic("uniqid: no valid lease")
	}
	return id
}

func (me *Component) GetServiceId() int64 {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return int64(me.serverId)
}

type ServerIds struct{ Data map[int]bool }

func (me *Component) GetServiceIds() map[int]bool {
	me.mu.RLock()
	defer me.mu.RUnlock()
	ids := make(map[int]bool, len(me.serverIds.Data))
	for id, present := range me.serverIds.Data {
		ids[id] = present
	}
	return ids
}

func (me *Component) dispatchEvents() {
	defer me.wg.Done()
	for {
		select {
		case evt := <-me.ch:
			select {
			case me.mainCh <- evt:
			default:
			}
			for _, ch := range me.chs {
				select {
				case ch <- evt:
				case <-me.ctx.Done():
					return
				}
			}
		case <-me.ctx.Done():
			return
		}
	}
}

func (me *Component) notify(evt *Event) bool {
	select {
	case me.ch <- evt:
		return true
	case <-me.ctx.Done():
		return false
	}
}

func (me *Component) mainloop() {
	defer me.wg.Done()
	for me.ctx.Err() == nil {
		err := me.watch()
		if me.ctx.Err() != nil {
			return
		}
		if err != nil && !me.notify(&Event{Type: TypeError, Error: err}) {
			return
		}
	}
}

func (me *Component) watch() (retErr error) {
	ctx, cancel := context.WithCancel(me.ctx)
	defer cancel()
	grantStarted := time.Now()
	grant, err := me.etcdCli.Grant(ctx, int64(me.config.LeaseTimeout/time.Second))
	if err != nil {
		return errors.WithMessage(err, "etcd.Grant")
	}
	defer func() {
		cancel()
		me.mu.Lock()
		me.leaseValid = false
		me.mu.Unlock()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), me.config.CloseTimeout)
		defer cleanupCancel()
		if _, err := me.etcdCli.Lease.Revoke(cleanupCtx, grant.ID); err != nil {
			err = errors.WithMessage(err, "etcd.Revoke")
			retErr = stderrors.Join(retErr, err)
			if me.ctx.Err() != nil {
				me.mu.Lock()
				me.closeErr = stderrors.Join(me.closeErr, err)
				me.mu.Unlock()
			}
		}
	}()
	if grant.TTL <= 0 {
		return errors.New("etcd.Grant returned invalid TTL")
	}
	leaseUntil := grantStarted.Add(time.Duration(grant.TTL) * time.Second)
	reply, err := me.etcdCli.Get(ctx, me.config.KeyPrefix, clientv3.WithPrefix())
	if err != nil {
		return errors.WithMessage(err, "etcd.Get")
	}
	if reply.Header == nil {
		return errors.New("etcd.Get returned no revision")
	}
	// 从快照之后的修订号观察，覆盖 Get 与 Watch 建立之间发生的变更。
	watcher := me.etcdCli.Watch(ctx, me.config.KeyPrefix, clientv3.WithPrefix(), clientv3.WithRev(reply.Header.Revision+1))
	serverIds := make(map[int]bool, len(reply.Kvs))
	for _, kv := range reply.Kvs {
		id, err := me.parseKey(string(kv.Key))
		if err == nil {
			serverIds[id] = true
		}
	}
	serverID := -1
	var claimRevision int64
	for i := me.config.MinId; i <= me.config.MaxId; i++ {
		if serverIds[i] {
			continue
		}
		resp, err := me.etcdCli.Txn(ctx).
			If(clientv3.Compare(clientv3.CreateRevision(fmt.Sprintf("%v%v", me.config.KeyPrefix, i)), "=", 0)).
			Then(clientv3.OpPut(fmt.Sprintf("%v%v", me.config.KeyPrefix, i), "", clientv3.WithLease(grant.ID))).Commit()
		if err != nil {
			return errors.WithMessage(err, "etcd.Txn")
		}
		if resp.Succeeded {
			if resp.Header == nil {
				return errors.New("etcd.Txn returned no revision")
			}
			serverID, claimRevision = i, resp.Header.Revision
			serverIds[i] = true
			break
		}
	}
	if serverID < 0 {
		return errors.New("no available serverId")
	}
	// 节点号可被另一实例立即复用；跨过一个完整毫秒后再发号，避免新序列重用上一任的时间片。
	handoffTimer := time.NewTimer(time.Millisecond)
	defer handoffTimer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-handoffTimer.C:
	}
	me.mu.Lock()
	if err := ctx.Err(); err != nil {
		me.mu.Unlock()
		return err
	}
	if me.closing || !time.Now().Before(leaseUntil) {
		me.mu.Unlock()
		return errors.New("lease expired before ID initialization")
	}
	// 同一节点号重新获得租约时保留序列计数，避免同一毫秒重置序列导致重复。
	if me.node == nil || me.serverId != serverID {
		node, err := snowflake.NewNode(int64(serverID))
		if err != nil {
			me.mu.Unlock()
			return errors.WithMessage(err, "snowflake.NewNode")
		}
		me.node = node
	}
	me.serverId, me.serverIds = serverID, &ServerIds{Data: serverIds}
	me.leaseUntil, me.leaseValid = leaseUntil, true
	me.mu.Unlock()
	if !me.notify(&Event{Type: TypeServerIdUpdate}) || !me.notify(&Event{Type: TypeServerIdsUpdate}) {
		return ctx.Err()
	}
	timer := time.NewTimer(time.Until(leaseUntil) / 3)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			// 单次续租记录请求开始时刻，不使用可能在 channel 中滞留的 TTL 响应延长本地有效期。
			renewStarted := time.Now()
			renewCtx, renewCancel := context.WithDeadline(ctx, leaseUntil)
			resp, err := me.etcdCli.KeepAliveOnce(renewCtx, grant.ID)
			renewCancel()
			if err != nil {
				return errors.WithMessage(err, "etcd.KeepAliveOnce")
			}
			if resp == nil || resp.ID != grant.ID || resp.TTL <= 0 {
				return errors.New("etcd.KeepAliveOnce returned invalid lease")
			}
			leaseUntil = renewStarted.Add(time.Duration(resp.TTL) * time.Second)
			if !time.Now().Before(leaseUntil) {
				return errors.New("lease expired during renewal")
			}
			me.mu.Lock()
			me.leaseUntil = leaseUntil
			me.mu.Unlock()
			timer.Reset(time.Until(leaseUntil) / 3)
		case ev, ok := <-watcher:
			if !ok {
				return errors.New("etcd.Watch closed")
			}
			if err := ev.Err(); err != nil {
				return errors.WithMessage(err, "etcd.Watch")
			}
			me.mu.Lock()
			ownershipLost := false
			for _, event := range ev.Events {
				id, err := me.parseKey(string(event.Kv.Key))
				if err != nil {
					continue
				}
				if id == serverID && event.Kv.ModRevision < claimRevision {
					continue
				}
				if event.Type == clientv3.EventTypePut {
					me.serverIds.Data[id] = true
					if id == serverID && event.Kv.Lease != int64(grant.ID) {
						ownershipLost = true
					}
				} else if event.Type == clientv3.EventTypeDelete {
					delete(me.serverIds.Data, id)
					if id == serverID {
						ownershipLost = true
					}
				}
			}
			me.mu.Unlock()
			if ownershipLost {
				return errors.New("server ID ownership lost")
			}
			if !me.notify(&Event{Type: TypeServerIdsUpdate}) {
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (me *Component) parseKey(key string) (int, error) {
	tmp, ok := strings.CutPrefix(key, me.config.KeyPrefix)
	if !ok {
		return 0, errors.Errorf("parseKey: %s", key)
	}
	if id, err := strconv.Atoi(tmp); err != nil {
		return 0, errors.WithMessage(err, "parseKey")
	} else {
		return id, nil
	}
}
