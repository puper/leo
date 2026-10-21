package reconnect

import (
	"testing"
	"time"
)

func TestBackoffRemainsBoundedAfterLongOutage(t *testing.T) {
	b := NewBackoff(&DefaultReconnectConfig{InitialInterval: time.Second, MaxInterval: 30 * time.Second, Multiplier: 2})
	for i := 0; i < 10000; i++ {
		d := b.NextDelay()
		if d <= 0 || d > 30*time.Second {
			t.Fatalf("attempt=%d delay=%v", i, d)
		}
	}
}
