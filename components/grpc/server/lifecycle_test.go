package server

import (
	"github.com/puper/leo/v2/components/grpc/server/config"
	"net"
	"runtime"
	"testing"
)

func TestBuilderImmediateCloseReleasesListener(t *testing.T) {
	old := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(old)
	for i := 0; i < 100; i++ {
		probe, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := probe.Addr().String()
		if err := probe.Close(); err != nil {
			t.Fatal(err)
		}
		value, err := Builder(&config.Config{Addr: addr})()
		if err != nil {
			t.Fatal(err)
		}
		if err := value.(*Component).Close(); err != nil {
			t.Fatal(err)
		}
		if err := value.(*Component).Close(); err != nil {
			t.Fatalf("repeat Close: %v", err)
		}
		replacement, err := net.Listen("tcp", addr)
		if err != nil {
			runtime.Gosched()
			t.Fatalf("iteration %d: Close returned nil but owned address is still in use: %v", i, err)
		}
		replacement.Close()
		runtime.Gosched()
	}
}
