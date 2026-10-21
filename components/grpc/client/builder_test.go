package client

import (
	"errors"
	"github.com/puper/leo/components/grpc/client/config"
	"google.golang.org/grpc/connectivity"
	"testing"
)

func TestBuilderClosesClientWhenConfigurerFails(t *testing.T) {
	want := errors.New("configuration failed")
	var captured *Component
	value, err := Builder(&config.Config{Addr: "127.0.0.1:1"}, func(c *Component) error { captured = c; return want })()
	if value != nil || !errors.Is(err, want) {
		t.Fatalf("value=%v error=%v", value, err)
	}
	defer captured.Close()
	if state := captured.GetState(); state != connectivity.Shutdown {
		t.Errorf("client still open: %v", state)
	}
}
