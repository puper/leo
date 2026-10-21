package uniqid

import (
	"context"
	"errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync/atomic"
	"testing"
	"time"
)

func expectLeasePanic(t *testing.T, c *Component) {
	t.Helper()
	defer func() {
		if got := recover(); got != "uniqid: no valid lease" {
			t.Errorf("unexpected panic: %v", got)
		}
	}()
	c.Generate()
	t.Error("generation accepted without a valid lease")
}
func TestGenerateRequiresLeaseAndProducesUniqueIDs(t *testing.T) {
	c, _, _ := testComponent(t)
	expectLeasePanic(t, c)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	seen := map[int64]bool{}
	for i := 0; i < 1000; i++ {
		id := c.Generate()
		if id.Node() != c.GetServiceId() || seen[id.Int64()] {
			t.Fatal("invalid or duplicate ID")
		}
		seen[id.Int64()] = true
	}
}
func oneLeaseThenBlock(l *testLease, ttl int64) {
	var grants atomic.Int32
	l.grantFn = func(ctx context.Context, _ int64) (*clientv3.LeaseGrantResponse, error) {
		if grants.Add(1) == 1 {
			return &clientv3.LeaseGrantResponse{ID: 1, TTL: ttl}, nil
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
}
func TestGenerateFailsWhenRenewalFails(t *testing.T) {
	c, l, _ := testComponent(t)
	oneLeaseThenBlock(l, 1)
	l.renewFn = func(context.Context, clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
		return nil, errors.New("renew failed")
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	deadline := time.Now().Add(2 * time.Second)
	for !l.revoked.Load() {
		if time.Now().After(deadline) {
			t.Fatal("failed lease was not revoked")
		}
		time.Sleep(time.Millisecond)
	}
	expectLeasePanic(t, c)
}
func TestGenerateFailsAtExpiryWithoutRenewalResponse(t *testing.T) {
	c, l, _ := testComponent(t)
	oneLeaseThenBlock(l, 1)
	l.renewFn = func(ctx context.Context, _ clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.mu.RLock()
	expiry := c.leaseUntil
	c.mu.RUnlock()
	time.Sleep(time.Until(expiry) + 10*time.Millisecond)
	expectLeasePanic(t, c)
}
func TestRenewalExtendsValidity(t *testing.T) {
	c, l, _ := testComponent(t)
	l.grantFn = func(context.Context, int64) (*clientv3.LeaseGrantResponse, error) {
		return &clientv3.LeaseGrantResponse{ID: 1, TTL: 1}, nil
	}
	l.renewFn = func(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
		return &clientv3.LeaseKeepAliveResponse{ID: id, TTL: 2}, nil
	}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.mu.RLock()
	initialExpiry := c.leaseUntil
	c.mu.RUnlock()
	time.Sleep(time.Until(initialExpiry) + 30*time.Millisecond)
	if c.Generate().Node() != 1 {
		t.Fatal("renewed lease did not produce valid ID")
	}
}
func TestOwnershipLossStopsGeneration(t *testing.T) {
	c, l, w := testComponent(t)
	oneLeaseThenBlock(l, 10)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	w.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("test/1"), ModRevision: 12}}}}
	deadline := time.Now().Add(time.Second)
	for !l.revoked.Load() {
		if time.Now().After(deadline) {
			t.Fatal("lost ownership was not released")
		}
		time.Sleep(time.Millisecond)
	}
	expectLeasePanic(t, c)
}
func TestHistoricalDeleteBeforeClaimDoesNotInvalidateLease(t *testing.T) {
	c, _, w := testComponent(t)
	sub, err := c.Subscribe(8)
	if err != nil {
		t.Fatal(err)
	}
	updates := sub.Events()
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	<-updates
	<-updates
	w.events <- clientv3.WatchResponse{Events: []*clientv3.Event{{Type: clientv3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("test/1"), ModRevision: 10}}}}
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("watch event not processed")
	}
	if !c.GetServiceIds()[1] || c.Generate().Node() != 1 {
		t.Fatal("historical event invalidated current claim")
	}
}
func TestCloseReportsRevokeFailure(t *testing.T) {
	c, l, _ := testComponent(t)
	want := errors.New("revoke failed")
	l.revokeFn = func(context.Context, clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) { return nil, want }
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); !errors.Is(err, want) {
		t.Fatalf("Close error=%v", err)
	}
	expectLeasePanic(t, c)
}
