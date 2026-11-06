package uniqid

import (
	"errors"
	"testing"
)

func TestGetErrorIsReturned(t *testing.T) {
	c, _, _ := testComponent(t)
	want := errors.New("get failed")
	c.etcdCli.KV = &testKV{getErr: want}
	if err := c.watch(); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	c.cancel()
}
