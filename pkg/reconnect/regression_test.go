package reconnect

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type generationConnector struct {
	mu                      sync.Mutex
	generation, disconnects int
	active                  bool
	badReconnect            bool
	ping                    func(context.Context) error
}

func (c *generationConnector) Connect(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active {
		c.badReconnect = true
	}
	c.generation++
	c.active = true
	return nil
}
func (c *generationConnector) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active {
		c.disconnects++
	}
	c.active = false
	return nil
}
func (c *generationConnector) IsConnected() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.active }
func (c *generationConnector) GetClient() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}
func (c *generationConnector) SendPing(ctx context.Context) error {
	if c.ping != nil {
		return c.ping(ctx)
	}
	return nil
}
func TestHealthReconnectReplacesExistingClient(t *testing.T) {
	fail := make(chan struct{})
	conn := &generationConnector{}
	conn.ping = func(context.Context) error {
		select {
		case <-fail:
			return errors.New("ping failed")
		default:
			return nil
		}
	}
	cfg := &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond, HealthCheckInterval: 10 * time.Millisecond}
	comp := New(conn, nil, cfg)
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	defer comp.Close()
	comp.WaitReconnect()
	client := comp.GetClient()
	fail <- struct{}{}
	deadline := time.After(time.Second)
	for {
		conn.mu.Lock()
		generation := conn.generation
		conn.mu.Unlock()
		if generation >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("no health reconnect")
		case <-time.After(time.Millisecond):
		}
	}
	if err := client.Do(func(raw interface{}) error {
		if raw.(int) < 2 {
			t.Errorf("stale client=%v", raw)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.badReconnect || conn.disconnects < 1 {
		t.Errorf("reconnected without closing old connection: bad=%v closes=%d", conn.badReconnect, conn.disconnects)
	}
}
func TestConcurrentClientRefreshAndRaw(t *testing.T) {
	conn := &generationConnector{}
	comp := New(conn, nil, &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond})
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	defer comp.Close()
	comp.WaitReconnect()
	client := comp.GetClient()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				client.Raw()
				_ = client.Do(func(raw interface{}) error {
					if raw.(int) < 4 {
						return errors.New("refresh")
					}
					return nil
				})
			}
		}()
	}
	wg.Wait()
}
func TestStartIsIdempotentAndCannotRestartAfterClose(t *testing.T) {
	conn := &generationConnector{}
	comp := New(conn, nil, nil)
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	comp.WaitReconnect()
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	comp.Close()
	if err := comp.Start(); err == nil {
		t.Fatal("restart after Close accepted")
	}
}
func TestCloseCancelsInflightHealthCheck(t *testing.T) {
	entered := make(chan struct{})
	conn := &generationConnector{}
	conn.ping = func(ctx context.Context) error { close(entered); <-ctx.Done(); return ctx.Err() }
	comp := New(conn, nil, &DefaultReconnectConfig{MaxRetries: -1, HealthCheckInterval: 400 * time.Millisecond})
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	<-entered
	done := make(chan struct{})
	go func() { comp.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("health check ignored shutdown cancellation")
		<-done
	}
}

func TestCloseTimeoutReportsBlockedConnector(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	conn := &mockConnector{connectFunc: func(context.Context) error {
		close(entered)
		<-release
		return nil
	}}
	comp := New(conn, nil, &DefaultReconnectConfig{CloseTimeout: 20 * time.Millisecond})
	if err := comp.Start(); err != nil {
		t.Fatal(err)
	}
	<-entered
	err := comp.Close()
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close error=%v", err)
	}
	select {
	case <-comp.doneCh:
	case <-time.After(time.Second):
		t.Fatal("connector did not exit after release")
	}
	if err := comp.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentStartAndClose(t *testing.T) {
	for i := 0; i < 50; i++ {
		comp := New(&mockConnector{}, nil, nil)
		done := make(chan struct{})
		go func() { _ = comp.Start(); close(done) }()
		if err := comp.Close(); err != nil {
			t.Fatal(err)
		}
		<-done
		if err := comp.Close(); err != nil {
			t.Fatal(err)
		}
		if comp.IsConnected() {
			t.Fatal("closed component remained connected")
		}
	}
}
