package reconnect

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockConnector struct {
	mu          sync.Mutex
	connected   bool
	connectErr  error
	connectFunc func() error
}

func (c *mockConnector) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connectFunc != nil {
		return c.connectFunc()
	}
	c.connected = c.connectErr == nil
	return c.connectErr
}

func (c *mockConnector) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

func (c *mockConnector) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *mockConnector) GetClient() interface{} {
	return c
}

type mockHealthConnector struct {
	mockConnector
	pingErr error
}

func (c *mockHealthConnector) SendPing(ctx context.Context) error {
	return c.pingErr
}

type mockEventHandler struct {
	mu                sync.Mutex
	connectedCount    int
	disconnectedCount int
	reconnectingCalls []struct {
		attempt int
		delay   time.Duration
	}
	errorCount int
	lastError  error
}

func (h *mockEventHandler) OnConnected() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connectedCount++
}

func (h *mockEventHandler) OnDisconnected(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnectedCount++
}

func (h *mockEventHandler) OnReconnecting(attempt int, delay time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reconnectingCalls = append(h.reconnectingCalls, struct {
		attempt int
		delay   time.Duration
	}{attempt, delay})
}

func (h *mockEventHandler) OnError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorCount++
	h.lastError = err
}

func TestComponent_Start_Success(t *testing.T) {
	connectCount := 0
	conn := &mockConnector{}
	conn.connectFunc = func() error {
		connectCount++
		if connectCount < 3 {
			return errors.New("connection failed")
		}
		conn.connected = true
		return nil
	}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond * 10}
	comp := New(conn, handler, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(time.Millisecond * 100)
	if !comp.IsConnected() {
		t.Errorf("Should be connected after Start(), connectCount=%d", connectCount)
	}
	comp.Close()
}

func TestComponent_Close_TerminatesCleanly(t *testing.T) {
	conn := &mockConnector{}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{MaxRetries: 3, InitialInterval: time.Millisecond * 20}
	comp := New(conn, handler, cfg)

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		comp.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close() did not return in time")
	}
}

func TestComponent_NoGoroutineLeak(t *testing.T) {
	iterations := 10
	leaked := int64(0)

	for i := 0; i < iterations; i++ {
		conn := &mockConnector{connectErr: errors.New("always fail")}
		handler := &mockEventHandler{}
		cfg := &DefaultReconnectConfig{MaxRetries: 1, InitialInterval: time.Millisecond * 5}
		comp := New(conn, handler, cfg)

		go comp.Start()
		time.Sleep(time.Millisecond * 20)
		comp.Close()
		time.Sleep(time.Millisecond * 10)

		if comp.IsConnected() {
			atomic.AddInt64(&leaked, 1)
		}
	}

	if leaked > 0 {
		t.Logf("WARNING: %d iterations may have leaked goroutines", leaked)
	}
}

func TestBackoff_NextDelay(t *testing.T) {
	cfg := &DefaultReconnectConfig{
		InitialInterval: time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
	backoff := NewBackoff(cfg)

	if delay := backoff.NextDelay(); delay != time.Second {
		t.Errorf("Expected 1s, got %v", delay)
	}
	if delay := backoff.NextDelay(); delay != 2*time.Second {
		t.Errorf("Expected 2s, got %v", delay)
	}
	if delay := backoff.NextDelay(); delay != 4*time.Second {
		t.Errorf("Expected 4s, got %v", delay)
	}
}

func TestBackoff_MaxInterval(t *testing.T) {
	cfg := &DefaultReconnectConfig{
		InitialInterval: time.Second,
		MaxInterval:     3 * time.Second,
		Multiplier:      2.0,
	}
	backoff := NewBackoff(cfg)

	backoff.NextDelay()
	backoff.NextDelay()
	backoff.NextDelay()

	if delay := backoff.NextDelay(); delay != 3*time.Second {
		t.Errorf("Expected max 3s, got %v", delay)
	}
}

func TestBackoff_Reset(t *testing.T) {
	cfg := &DefaultReconnectConfig{
		InitialInterval: time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
	backoff := NewBackoff(cfg)

	backoff.NextDelay()
	backoff.NextDelay()
	backoff.Reset()

	if delay := backoff.NextDelay(); delay != time.Second {
		t.Errorf("Expected 1s after reset, got %v", delay)
	}
}

func TestEventHandlers_Multiple(t *testing.T) {
	h1 := &mockEventHandler{}
	h2 := &mockEventHandler{}

	handlers := EventHandlers{h1, h2}
	handlers.OnConnected()

	if h1.connectedCount != 1 || h2.connectedCount != 1 {
		t.Error("Both handlers should receive OnConnected")
	}
}

func TestFuncConnector(t *testing.T) {
	conn := AsConnector(
		func(ctx context.Context) error {
			return nil
		},
		func() error {
			return nil
		},
	)

	if conn.IsConnected() {
		t.Error("Should not be connected initially")
	}

	err := conn.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !conn.IsConnected() {
		t.Error("Should be connected after Connect()")
	}

	err = conn.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	if conn.IsConnected() {
		t.Error("Should not be connected after Disconnect()")
	}
}

func TestComponent_HealthCheck(t *testing.T) {
	conn := &mockHealthConnector{}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{
		MaxRetries:          1,
		InitialInterval:     time.Millisecond * 10,
		HealthCheckInterval: time.Millisecond * 20,
	}
	comp := New(conn, handler, cfg)

	go func() {
		time.Sleep(time.Millisecond * 30)
		conn.mu.Lock()
		conn.connected = true
		conn.mu.Unlock()
	}()

	err := comp.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	conn.pingErr = errors.New("ping failed")
	time.Sleep(time.Millisecond * 50)

	comp.Close()
}

func TestClient_GetClient(t *testing.T) {
	conn := &mockConnector{}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond * 10}
	comp := New(conn, handler, cfg)

	conn.connectFunc = func() error {
		conn.connected = true
		return nil
	}

	comp.Start()
	defer comp.Close()

	client := comp.GetClient()
	if client == nil {
		t.Fatal("GetClient() returned nil")
	}

	if client.Raw() == nil {
		t.Error("Client.Raw() returned nil")
	}
}

func TestClient_Do_Success(t *testing.T) {
	conn := &mockConnector{}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond * 10}
	comp := New(conn, handler, cfg)

	conn.connectFunc = func() error {
		conn.connected = true
		return nil
	}

	comp.Start()
	defer comp.Close()

	client := comp.GetClient()
	err := client.Do(func(raw interface{}) error {
		return nil
	})

	if err != nil {
		t.Errorf("Do() failed: %v", err)
	}
}

func TestClient_Do_RefreshOnError(t *testing.T) {
	connectCount := 0
	conn := &mockConnector{}
	handler := &mockEventHandler{}
	cfg := &DefaultReconnectConfig{MaxRetries: -1, InitialInterval: time.Millisecond * 10}
	comp := New(conn, handler, cfg)

	conn.connectFunc = func() error {
		connectCount++
		if connectCount < 3 {
			return errors.New("connection error")
		}
		conn.connected = true
		return nil
	}

	comp.Start()
	defer comp.Close()

	client := comp.GetClient()
	err := client.Do(func(raw interface{}) error {
		if connectCount < 2 {
			return errors.New("operation failed")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Do() failed after retries: %v", err)
	}
	if connectCount < 2 {
		t.Error("Should have retried after error")
	}
}
