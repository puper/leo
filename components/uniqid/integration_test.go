package uniqid

import (
	"context"
	"errors"
	"fmt"
	"github.com/puper/leo/v2/components/uniqid/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"testing"
	"time"
)

func TestEtcdLeaseAllocationRenewalAndRelease(t *testing.T) {
	endpoint := os.Getenv("LEO_TEST_ETCD_ENDPOINT")
	if endpoint == "" {
		t.Skip("requires isolated etcd via LEO_TEST_ETCD_ENDPOINT")
	}
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{endpoint}, DialTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	prefix := fmt.Sprintf("leo-audit/%d/", time.Now().UnixNano())
	cfg := &config.Config{KeyPrefix: prefix, MinId: 1, MaxId: 2, LeaseTimeout: 2 * time.Second, InitTimeout: 3 * time.Second, CloseTimeout: 2 * time.Second}
	first, second := New(cfg), New(cfg)
	first.etcdCli = cli
	second.etcdCli = cli
	slow, err := first.Subscribe(2)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	defer second.Close()
	if err = first.Start(); err != nil {
		t.Fatal(err)
	}
	if err = second.Start(); err != nil {
		t.Fatal(err)
	}
	if first.GetServiceId() == second.GetServiceId() {
		t.Fatal("instances claimed same node")
	}
	first.mu.RLock()
	initialExpiry := first.leaseUntil
	first.mu.RUnlock()
	time.Sleep(time.Until(initialExpiry) + 100*time.Millisecond)
	if !errors.Is(slow.Err(), ErrSlowSubscriber) {
		t.Fatalf("node changes did not terminate slow subscriber: %v", slow.Err())
	}
	seen := map[int64]bool{}
	for i := 0; i < 100; i++ {
		for _, c := range []*Component{first, second} {
			id := c.Generate()
			if seen[id.Int64()] {
				t.Fatal("duplicate ID")
			}
			seen[id.Int64()] = true
		}
	}
	key := fmt.Sprintf("%s%d", prefix, first.GetServiceId())
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	expectLeasePanic(t, first)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	reply, err := cli.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Kvs) != 0 {
		t.Fatal("lease key remained after Close")
	}
	if second.Generate().Node() != second.GetServiceId() {
		t.Fatal("closing first affected second lease")
	}
	// 释放后的节点号由新实例连续接管，覆盖真实 etcd 下的跨实例交接。
	for i := 0; i < 20; i++ {
		next := New(cfg)
		next.etcdCli = cli
		defer next.Close()
		if err := next.Start(); err != nil {
			t.Fatal(err)
		}
		id := next.Generate()
		if err := next.Close(); err != nil {
			t.Fatal(err)
		}
		if id.Node() != first.GetServiceId() {
			t.Fatal("new instance did not reclaim the released node")
		}
		if seen[id.Int64()] {
			t.Fatal("duplicate ID across lease handoff")
		}
		seen[id.Int64()] = true
	}
}
