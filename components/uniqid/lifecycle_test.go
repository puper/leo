package uniqid

import (
	"context"
	"github.com/puper/leo/components/uniqid/config"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync/atomic"
	"testing"
	"time"
)

type testKV struct {
	clientv3.KV
	getErr error
}

func (k *testKV) Get(context.Context, string, ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if k.getErr != nil {
		return nil, k.getErr
	}
	return &clientv3.GetResponse{Header: &pb.ResponseHeader{Revision: 10}}, nil
}
func (k *testKV) Txn(context.Context) clientv3.Txn { return &testTxn{} }

type testTxn struct{ clientv3.Txn }

func (t *testTxn) If(...clientv3.Cmp) clientv3.Txn  { return t }
func (t *testTxn) Then(...clientv3.Op) clientv3.Txn { return t }
func (t *testTxn) Commit() (*clientv3.TxnResponse, error) {
	return &clientv3.TxnResponse{Succeeded: true, Header: &pb.ResponseHeader{Revision: 11}}, nil
}

type testLease struct {
	clientv3.Lease
	revoked  atomic.Bool
	grantFn  func(context.Context, int64) (*clientv3.LeaseGrantResponse, error)
	renewFn  func(context.Context, clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error)
	revokeFn func(context.Context, clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error)
}

func (l *testLease) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	if l.grantFn != nil {
		return l.grantFn(ctx, ttl)
	}
	return &clientv3.LeaseGrantResponse{ID: 1, TTL: 10}, nil
}
func (l *testLease) Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	l.revoked.Store(true)
	if l.revokeFn != nil {
		return l.revokeFn(ctx, id)
	}
	return &clientv3.LeaseRevokeResponse{}, nil
}
func (l *testLease) KeepAliveOnce(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
	if l.renewFn != nil {
		return l.renewFn(ctx, id)
	}
	return &clientv3.LeaseKeepAliveResponse{ID: id, TTL: 10}, nil
}

type testWatcher struct {
	clientv3.Watcher
	events   chan clientv3.WatchResponse
	revision atomic.Int64
}

func (w *testWatcher) Watch(_ context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	w.revision.Store(clientv3.OpGet(key, opts...).Rev())
	return w.events
}
func testComponent(t *testing.T) (*Component, *testLease, *testWatcher) {
	t.Helper()
	lease := &testLease{}
	watcher := &testWatcher{events: make(chan clientv3.WatchResponse)}
	c := New(&config.Config{LeaseTimeout: 10 * time.Second, InitTimeout: time.Second, CloseTimeout: time.Second, KeyPrefix: "test/", MinId: 1, MaxId: 8})
	c.etcdCli = &clientv3.Client{KV: &testKV{}, Lease: lease, Watcher: watcher}
	return c, lease, watcher
}
func TestServiceIDsAreSnapshots(t *testing.T) {
	c, _, _ := testComponent(t)
	c.serverIds.Data[1] = true
	ids := c.GetServiceIds()
	delete(ids, 1)
	if !c.GetServiceIds()[1] {
		t.Fatal("caller mutated internal IDs")
	}
}
func TestCloseStopsSlowSubscriber(t *testing.T) {
	c, _, _ := testComponent(t)
	subscriber := make(chan *Event)
	c.chs = []chan *Event{subscriber}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-subscriber:
		t.Error("dispatcher remained blocked after Close")
	case <-time.After(30 * time.Millisecond):
	}
	// 排空旧实现遗留的发送，避免失败的回归夹具留下后台任务。
	deadline := time.NewTimer(50 * time.Millisecond)
	defer deadline.Stop()
	for {
		select {
		case <-subscriber:
		case <-deadline.C:
			return
		}
	}
}
func TestClosedWatchReturnsError(t *testing.T) {
	c, _, w := testComponent(t)
	close(w.events)
	go func() {
		for {
			select {
			case <-c.ch:
			case <-c.ctx.Done():
				return
			}
		}
	}()
	done := make(chan error, 1)
	go func() { done <- c.watch() }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("closed watch accepted")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("closed watch spun without returning")
		c.cancel()
		<-done
	}
	c.cancel()
}
func TestWatchStartsAfterSnapshotRevision(t *testing.T) {
	c, _, w := testComponent(t)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if got := w.revision.Load(); got != 11 {
		t.Errorf("watch revision=%d want=11", got)
	}
}
func TestConcurrentServiceIDReads(t *testing.T) {
	c, _, w := testComponent(t)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			_ = c.GetServiceId()
			_ = c.GetServiceIds()
		}
	}()
	for i := 0; i < 100; i++ {
		w.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypePut, Kv: &mvccpb.KeyValue{Key: []byte("test/2")}}}}
	}
	<-done
}
func TestGenerateDoesNotContinueAfterLeaseRevoke(t *testing.T) {
	c, l, _ := testComponent(t)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !l.revoked.Load() {
		t.Fatal("lease was not revoked")
	}
	expectLeasePanic(t, c)
}
