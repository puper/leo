package uniqid

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/uniqid/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Component struct {
	*snowflake.Node
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

	closing int64
}

func New(cfg *config.Config) *Component {
	me := &Component{
		config: cfg,
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
	if me.etcdCli == nil {
		return errors.New("etcd client is nil")
	}
	me.wg.Add(1)
	go me.dispatchEvents()
	go me.mainloop()
	var err error
	for {
		select {
		case evt := <-me.mainCh:
			if evt.Type == TypeServerIdUpdate {
				me.Node, err = snowflake.NewNode(int64(me.serverId))
				if err != nil {
					return errors.WithMessage(err, "snowflake.NewNode")
				}
				return nil
			} else if evt.Type == TypeError {
				me.Close()
				return evt.Error
			}
		case <-time.After(me.config.InitTimeout):
			me.Close()
			return errors.New("New: timeout")
		}
	}
}

func (me *Component) Close() error {
	if atomic.LoadInt64(&me.closing) > 0 {
		return nil
	}
	atomic.StoreInt64(&me.closing, 1)
	me.cancel()
	select {
	case <-time.After(me.config.CloseTimeout):
		return errors.New("Close: timeout")
	case <-func() chan struct{} {
		ch := make(chan struct{})
		go func() {
			me.wg.Wait()
			close(ch)
		}()
		return ch
	}():
	}
	return nil
}

func (me *Component) GetServiceId() int64 {
	return int64(me.serverId)
}

type ServerIds struct {
	Data map[int]bool
}

func (me *Component) GetServiceIds() map[int]bool {
	return me.serverIds.Data
}

func (me *Component) dispatchEvents() {
	for {
		select {
		case evt := <-me.ch:
			if evt.Type == TypeServerIdUpdate {
				n, err := snowflake.NewNode(int64(me.serverId))
				if err == nil {
					me.Node = n
				}
			}
			select {
			case me.mainCh <- evt:
			default:
			}
			for _, ch := range me.chs {
				ch <- evt
			}
		case <-me.ctx.Done():
			return
		}
	}
}

func (me *Component) mainloop() {
	func() {
		defer me.wg.Done()
		for {
			err := me.watch()
			if err != nil {
				select {
				case <-me.ctx.Done():
					return
				default:
					me.ch <- &Event{Type: TypeError, Error: err}
				}
			}
		}

	}()
}

func (me *Component) watch() error {
	ctx, cancel := context.WithCancel(me.ctx)
	defer cancel()
	grant, err := me.etcdCli.Grant(ctx, int64(me.config.LeaseTimeout/time.Second))
	if err != nil {
		return errors.WithMessage(err, "etcd.Grant")
	}
	defer me.etcdCli.Lease.Revoke(context.TODO(), clientv3.LeaseID(grant.ID))
	watcher := me.etcdCli.Watch(ctx, me.config.KeyPrefix, clientv3.WithPrefix())
	reply, _ := me.etcdCli.Get(ctx, me.config.KeyPrefix, clientv3.WithPrefix())
	serverIds := make(map[int]bool, len(reply.Kvs))
	for _, kv := range reply.Kvs {
		id, err := me.parseKey(string(kv.Key))
		if err != nil {
			continue
		}
		serverIds[id] = true
	}
	found := false
	for i := me.config.MinId; i <= me.config.MaxId; i++ {
		if _, ok := serverIds[i]; !ok {
			resp, err := me.etcdCli.Txn(ctx).If(clientv3.Compare(clientv3.CreateRevision(fmt.Sprintf("%v%v", me.config.KeyPrefix, i)), "=", 0)).
				Then(clientv3.OpPut(fmt.Sprintf("%v%v", me.config.KeyPrefix, i), "", clientv3.WithLease(grant.ID))).
				Commit()
			if err != nil {
				return errors.WithMessage(err, "etcd.Txn")
			}
			if resp.Succeeded {
				me.serverId = i
				serverIds[i] = true
				found = true
				break
			}
		}
	}
	if !found {
		return fmt.Errorf("no available serverId")
	}
	me.serverIds = &ServerIds{Data: serverIds}
	me.ch <- &Event{Type: TypeServerIdUpdate}
	me.ch <- &Event{Type: TypeServerIdsUpdate}
	keepAliveResp, err := me.etcdCli.KeepAlive(ctx, grant.ID)
	if err != nil {
		return errors.WithMessage(err, "etcd.KeepAlive")
	}
	for {
		select {
		case _, ok := <-keepAliveResp:
			if !ok {
				return errors.New("etcd.KeepAlive")
			}
		case ev := <-watcher:
			serverIds := map[int]bool{}
			for serverId := range me.serverIds.Data {
				serverIds[serverId] = true
			}
			for _, e := range ev.Events {
				id, err := me.parseKey(string(e.Kv.Key))
				if err != nil {
					continue
				}
				if e.Type == clientv3.EventTypePut {
					serverIds[id] = true
				} else if e.Type == clientv3.EventTypeDelete {
					delete(serverIds, id)
				}
			}
			me.serverIds = &ServerIds{Data: serverIds}
			me.ch <- &Event{Type: TypeServerIdsUpdate}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return errors.WithMessage(err, "etcd.Context")
			} else {
				return nil
			}
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
