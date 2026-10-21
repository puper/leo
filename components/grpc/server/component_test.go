package server

import (
	"context"
	"errors"
	"github.com/puper/leo/v2/components/grpc/protos"
	"github.com/puper/leo/v2/components/grpc/server/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
	"time"
)

type waitingHealth struct {
	protos.UnimplementedAnyServiceServer
	entered chan struct{}
}

func (h *waitingHealth) Call(ctx context.Context, _ *protos.AnyRequest) (*protos.AnyReply, error) {
	close(h.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}
func TestCloseHonorsShutdownTimeout(t *testing.T) {
	srv := grpc.NewServer()
	defer srv.Stop()
	h := &waitingHealth{entered: make(chan struct{})}
	protos.RegisterAnyServiceServer(srv, h)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()
	go srv.Serve(lis)
	cli, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rpcDone := make(chan struct{})
	go func() { defer close(rpcDone); _, _ = protos.NewAnyServiceClient(cli).Call(ctx, &protos.AnyRequest{}) }()
	select {
	case <-h.entered:
	case <-ctx.Done():
		t.Fatal("RPC did not start")
	}
	c := &Component{Server: srv, config: &config.Config{ShutdownTimeout: 30 * time.Millisecond}}
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case err := <-closed:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("error=%v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Error("Close exceeded configured timeout")
		srv.Stop()
		<-closed
	}
	select {
	case <-rpcDone:
	case <-ctx.Done():
		t.Error("RPC was not cancelled")
	}
}
func TestCloseWithoutActiveRPC(t *testing.T) {
	c := &Component{Server: grpc.NewServer(), config: &config.Config{ShutdownTimeout: time.Second}}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
