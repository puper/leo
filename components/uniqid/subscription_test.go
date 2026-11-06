package uniqid

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestSubscriberDoesNotStopRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		c, lease, watcher := testComponent(t)
		slow, err := c.Subscribe(2)
		if err != nil {
			t.Fatal(err)
		}
		active, err := c.Subscribe(4)
		if err != nil {
			t.Fatal(err)
		}
		oneLeaseThenBlock(lease, 1)
		var renewals atomic.Int32
		lease.renewFn = func(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
			renewals.Add(1)
			return &clientv3.LeaseKeepAliveResponse{ID: id, TTL: 1}, nil
		}
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		for _, want := range []EventType{TypeServerIdUpdate, TypeServerIdsUpdate} {
			if got := <-active.Events(); got.Type != want {
				t.Fatalf("event=%v want=%v", got.Type, want)
			}
		}
		c.mu.RLock()
		initialExpiry := c.leaseUntil
		c.mu.RUnlock()
		for i := 0; i < 3; i++ {
			watcher.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypePut, Kv: &mvccpb.KeyValue{Key: []byte("test/2"), ModRevision: int64(12 + i)}}}}
			if evt := <-active.Events(); evt.Type != TypeServerIdsUpdate {
				t.Fatalf("event=%v", evt)
			}
		}
		if !errors.Is(slow.Err(), ErrSlowSubscriber) {
			t.Fatalf("slow subscriber error=%v", slow.Err())
		}
		if err := active.Err(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Until(initialExpiry) + time.Millisecond)
		synctest.Wait()
		if renewals.Load() == 0 {
			t.Fatal("slow subscriber prevented renewal")
		}
		if c.Generate().Node() != 1 {
			t.Fatal("renewed lease not available")
		}
		// 即使订阅溢出，节点删除仍须由同一观察循环处理并停止发号。
		watcher.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("test/1"), ModRevision: 15}}}}
		synctest.Wait()
		expectLeasePanic(t, c)
		evt := <-active.Events()
		if evt.Type != TypeError || evt.Error == nil {
			t.Fatalf("ownership failure event=%v", evt)
		}
	})
}

func TestSubscriptionOverflowPreservesQueuedEvents(t *testing.T) {
	c, _, _ := testComponent(t)
	defer c.Close()
	sub, err := c.Subscribe(2)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("lease failure")
	c.notify(Event{Type: TypeServerIdUpdate})
	c.notify(Event{Type: TypeError, Error: failure})
	c.notify(Event{Type: TypeServerIdsUpdate})
	if !errors.Is(sub.Err(), ErrSlowSubscriber) {
		t.Fatalf("overflow error=%v", sub.Err())
	}
	var events []Event
	for evt := range sub.Events() {
		events = append(events, evt)
	}
	if len(events) != 2 || events[0].Type != TypeServerIdUpdate || !errors.Is(events[1].Error, failure) {
		t.Fatalf("queued events changed: %v", events)
	}
	if !errors.Is(sub.Close(), ErrSlowSubscriber) {
		t.Fatal("Close cleared overflow error")
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	c, _, _ := testComponent(t)
	for _, capacity := range []int{0, -1} {
		if sub, err := c.Subscribe(capacity); err == nil || sub != nil {
			t.Fatalf("invalid capacity %d accepted", capacity)
		}
	}
	cancelled, err := c.Subscribe(1)
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := c.Subscribe(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := cancelled.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cancelled.Close(); err != nil {
		t.Fatal(err)
	}
	c.notify(Event{Type: TypeServerIdUpdate})
	if _, ok := <-cancelled.Events(); ok {
		t.Fatal("cancelled subscriber received an event")
	}
	if evt := <-remaining.Events(); evt.Type != TypeServerIdUpdate {
		t.Fatal(evt)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-remaining.Events(); ok {
		t.Fatal("component Close did not close subscription")
	}
	if err := remaining.Err(); err != nil {
		t.Fatal(err)
	}
	if sub, err := c.Subscribe(1); err == nil || sub != nil {
		t.Fatal("subscribed after component Close")
	}
}

func TestStartupFailureBypassesFullSubscription(t *testing.T) {
	c, _, _ := testComponent(t)
	want := errors.New("snapshot unavailable")
	c.etcdCli.KV = &testKV{getErr: want}
	sub, err := c.Subscribe(1)
	if err != nil {
		t.Fatal(err)
	}
	c.notify(Event{Type: TypeServerIdsUpdate})
	if err := c.Start(); !errors.Is(err, want) {
		t.Fatalf("startup error=%v", err)
	}
	for range sub.Events() {
	}
	select {
	case <-c.done:
	default:
		t.Fatal("failed Start did not stop lease worker")
	}
}

func TestBuilderFailureClosesSubscriptions(t *testing.T) {
	for _, configureFailure := range []bool{false, true} {
		var sub *Subscription
		want := errors.New("configuration failed")
		value, err := Builder(nil, func(c *Component) error {
			var err error
			sub, err = c.Subscribe(1)
			if err != nil {
				return err
			}
			if configureFailure {
				return want
			}
			return nil
		})()
		if value != nil || err == nil {
			t.Fatal("failed Builder succeeded")
		}
		if configureFailure && !errors.Is(err, want) {
			t.Fatal(err)
		}
		select {
		case _, ok := <-sub.Events():
			if ok {
				t.Fatal("unexpected event")
			}
		default:
			t.Fatal("failed Builder left subscription open")
		}
	}
}

func TestConcurrentSubscribePublishAndClose(t *testing.T) {
	c, _, _ := testComponent(t)
	var wg sync.WaitGroup
	var ready sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		ready.Add(1)
		go func() {
			defer wg.Done()
			first, err := c.Subscribe(4)
			if err != nil {
				t.Error(err)
			}
			ready.Done()
			<-start
			if first == nil {
				return
			}
			c.notify(Event{Type: TypeServerIdsUpdate})
			_ = first.Close()
			for j := 0; j < 100; j++ {
				sub, err := c.Subscribe(4)
				if err != nil {
					return
				}
				c.notify(Event{Type: TypeServerIdsUpdate})
				_ = sub.Err()
				_ = sub.Close()
				for range sub.Events() {
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			c.notify(Event{Type: TypeServerIdUpdate})
		}
	}()
	ready.Wait()
	close(start)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if len(c.subscriptions) != 0 {
		t.Fatal("closed component retained subscriptions")
	}
}
