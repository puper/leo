package etcd

import (
	"errors"
	"github.com/puper/leo/v2/components/etcd/config"
	"testing"
	"time"
)

func TestBuilderClosesClientWhenConfigurerFails(t *testing.T) {
	want := errors.New("configuration failed")
	var captured *Component
	value, err := Builder(&config.Config{Endpoints: []string{"127.0.0.1:1"}, DialTimeout: time.Millisecond}, func(c *Component) error { captured = c; return want })()
	if value != nil || !errors.Is(err, want) {
		t.Fatalf("value=%v error=%v", value, err)
	}
	defer captured.Close()
	select {
	case <-captured.Ctx().Done():
	default:
		t.Error("failed builder left client open")
	}
}
