package subscription

import (
	"errors"
	"github.com/puper/leo/components/rabbitmq/subscription/config"
	"io"
	"net"
	"testing"
	"time"
)

func TestFailedStartStopsSubscription(t *testing.T) {
	sub := New(&config.Config{Addr: "invalid://address", StartTimeout: time.Second, CloseTimeout: time.Second})
	defer sub.Close()
	if err := sub.Start(); err == nil {
		t.Fatal("invalid address accepted")
	}
	select {
	case <-sub.ctx.Done():
	default:
		t.Error("failed Start left reconnect worker running")
	}
}
func TestCloseCancelsAMQPHandshake(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, e := lis.Accept()
		if e == nil {
			accepted <- c
		}
	}()
	sub := New(&config.Config{Addr: "amqp://" + lis.Addr().String() + "/", StartTimeout: 4 * time.Second, CloseTimeout: 3 * time.Second})
	started := make(chan error, 1)
	go func() { started <- sub.Start() }()
	var peer net.Conn
	select {
	case peer = <-accepted:
	case <-time.After(time.Second):
		t.Fatal("no connection")
	}
	defer peer.Close()
	if err = peer.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadFull(peer, make([]byte, 8)); err != nil {
		t.Fatal(err)
	}
	closed := make(chan error, 1)
	go func() { closed <- sub.Close() }()
	select {
	case err = <-closed:
		if err != nil {
			t.Errorf("normal Close=%v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Error("Close did not cancel handshake")
		peer.Close()
		<-closed
	}
	select {
	case err = <-started:
		if err == nil {
			t.Error("cancelled Start succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("Start remained blocked")
	}
	if err = sub.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("repeat Close=%v", err)
	}
}
