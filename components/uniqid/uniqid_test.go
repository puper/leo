package uniqid

import (
	"testing"
)

func TestEtcdGetNilPanic(t *testing.T) {
	t.Log("BUG-003 DOCUMENTATION TEST")
	t.Log("Location: components/uniqid/component.go:160")
	t.Log("Issue: etcdCli.Get() error is ignored with _")
	t.Log("If Get() fails and returns nil, accessing reply.Kvs causes nil pointer panic")
	t.Log("")
	t.Log("The bug is in this code:")
	t.Log("  reply, _ := me.etcdCli.Get(ctx, me.config.KeyPrefix, clientv3.WithPrefix())")
	t.Log("  serverIds := make(map[int]bool, len(reply.Kvs)) // PANIC if reply is nil!")
	t.Log("")
	t.Log("Reproduction:")
	t.Log("1. Start uniqid component with etcd")
	t.Log("2. Simulate network failure during watch() Get() call")
	t.Log("3. reply becomes nil, reply.Kvs panics")

	t.Skip("This test requires a running etcd instance to reproduce the actual panic")
}
