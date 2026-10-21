package uniqid

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestSequentialLeaseHandoffKeepsIDsUnique(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 将虚拟时间移至 Snowflake epoch 之后，使同毫秒交接不依赖实际调度速度。
		time.Sleep(time.Until(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		var mu sync.Mutex
		var active, next clientv3.LeaseID
		lease := &testLease{}
		lease.grantFn = func(context.Context, int64) (*clientv3.LeaseGrantResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			if active != 0 {
				return nil, errors.New("previous lease is still active")
			}
			next++
			active = next
			return &clientv3.LeaseGrantResponse{ID: active, TTL: 10}, nil
		}
		lease.revokeFn = func(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			if active != id {
				return nil, errors.New("wrong lease revoked")
			}
			active = 0
			return &clientv3.LeaseRevokeResponse{}, nil
		}
		seen := make(map[int64]int)
		for i := 0; i < 100; i++ {
			c, _, _ := testComponent(t)
			c.etcdCli.Lease = lease
			if err := c.Start(); err != nil {
				t.Fatal(err)
			}
			id := c.Generate().Int64()
			if err := c.Close(); err != nil {
				t.Fatal(err)
			}
			if prior, exists := seen[id]; exists {
				t.Fatalf("duplicate ID after lease handoff: instances %d and %d", prior, i)
			}
			seen[id] = i
		}
	})
}

func TestClaimWaitsBeforeGenerating(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		c, _, _ := testComponent(t)
		defer c.Close()
		started := make(chan error, 1)
		go func() { started <- c.Start() }()
		synctest.Wait()
		expectLeasePanic(t, c)
		select {
		case err := <-started:
			t.Fatalf("Start returned before the handoff delay: %v", err)
		default:
		}
		time.Sleep(time.Millisecond - time.Nanosecond)
		synctest.Wait()
		expectLeasePanic(t, c)
		select {
		case err := <-started:
			t.Fatalf("Start returned before a full millisecond: %v", err)
		default:
		}
		time.Sleep(time.Nanosecond)
		if err := <-started; err != nil {
			t.Fatal(err)
		}
		if c.Generate().Node() != c.GetServiceId() {
			t.Fatal("claimed node did not become available")
		}
	})
}

func TestCloseDuringClaimWaitStopsStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, lease, _ := testComponent(t)
		defer c.Close()
		started := make(chan error, 1)
		go func() { started <- c.Start() }()
		synctest.Wait()
		closedAt := time.Now()
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-started; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled Start error=%v", err)
		}
		if time.Since(closedAt) != 0 {
			t.Fatal("Close waited for the handoff timer instead of cancelling it")
		}
		if !lease.revoked.Load() {
			t.Fatal("cancelled claim left its lease active")
		}
		expectLeasePanic(t, c)
	})
}

func TestReclaimWaitsBeforeGenerating(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		c, _, watcher := testComponent(t)
		defer c.Close()
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		previous := c.Generate()
		watcher.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("test/1"), ModRevision: 12}}}}
		synctest.Wait()
		expectLeasePanic(t, c)
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if id := c.Generate(); id.Node() != previous.Node() || id <= previous {
			t.Fatal("reclaimed node did not resume with a newer ID")
		}
	})
}

type delayedSnapshotKV struct {
	*testKV
	delay time.Duration
}

func (k *delayedSnapshotKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	time.Sleep(k.delay)
	return k.testKV.Get(ctx, key, opts...)
}

func TestLeaseExpiryDuringClaimWaitPreventsPublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, lease, _ := testComponent(t)
		defer c.Close()
		c.config.InitTimeout = 2 * time.Second
		oneLeaseThenBlock(lease, 1)
		c.etcdCli.KV = &delayedSnapshotKV{testKV: &testKV{}, delay: time.Second - time.Millisecond/2}
		if err := c.Start(); err == nil || !strings.Contains(err.Error(), "lease expired before ID initialization") {
			t.Fatalf("expired claim Start error=%v", err)
		}
		if !lease.revoked.Load() {
			t.Fatal("expired claim was not revoked")
		}
		expectLeasePanic(t, c)
	})
}
