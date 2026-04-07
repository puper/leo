package reconnect

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Component struct {
	connector    Connector
	eventHandler EventHandler
	config       ReconnectConfig
	backoff      *Backoff

	lifecycleMu sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	mu        sync.Mutex
	cond      *sync.Cond
	connected bool
	closing   bool
	stopped   bool
	clientSeq int64

	doneCh chan struct{}
}

type Client struct {
	raw  interface{}
	seq  int64
	comp *Component
}

func New(connector Connector, eventHandler EventHandler, config ReconnectConfig) *Component {
	if eventHandler == nil {
		eventHandler = &NopEventHandler{}
	}

	if config == nil {
		if cp, ok := connector.(ConfigProvider); ok {
			config = cp.Config()
		}
	}

	if config == nil {
		config = &DefaultReconnectConfig{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Component{
		connector:    connector,
		eventHandler: eventHandler,
		config:       config,
		backoff:      NewBackoff(config),
		ctx:          ctx,
		cancel:       cancel,
		clientSeq:    0,
		doneCh:       make(chan struct{}),
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *Component) Start() error {
	c.mu.Lock()
	c.closing = false
	c.stopped = false
	c.mu.Unlock()
	c.wg.Add(1)
	go c.run()
	return nil
}

func (c *Component) Close() error {
	c.mu.Lock()
	c.closing = true
	c.connected = false
	c.mu.Unlock()
	c.cond.Broadcast()
	c.cancelContext()
	c.wg.Wait()
	return nil
}

func (c *Component) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *Component) GetClient() *Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	seq := atomic.LoadInt64(&c.clientSeq)
	var raw interface{}
	if cg, ok := c.connector.(ClientGetter); ok {
		raw = cg.GetClient()
	}

	return &Client{
		raw:  raw,
		seq:  seq,
		comp: c,
	}
}

func (c *Component) WaitReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for !c.connected && !c.stopped && !c.closing {
		c.cond.Wait()
	}
}

func (c *Component) WaitRefresh() {
	for {
		seq := atomic.LoadInt64(&c.clientSeq)
		if seq >= 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (c *Component) run() {
	defer c.wg.Done()
	defer c.backoff.Reset()
	defer close(c.doneCh)
	defer func() {
		c.mu.Lock()
		c.stopped = true
		c.connected = false
		c.mu.Unlock()
		c.cond.Broadcast()
	}()

	c.connectLoop()
}

func (c *Component) connectLoop() {
	attempt := 0
	for {
		if !c.waitRetry(attempt) {
			return
		}
		ctx := c.getContext()
		err := c.connector.Connect(ctx)
		if err == nil {
			c.backoff.Reset()
			attempt = 0
			c.setConnected(true)
			c.eventHandler.OnConnected()

			c.waitForDisconnect(ctx)

			c.setConnected(false)

			if c.getContext().Err() != nil {
				c.connector.Disconnect()
				return
			}
			c.eventHandler.OnDisconnected(nil)
			attempt = 1
		} else {
			c.eventHandler.OnError(err)
			attempt++
		}
	}
}

func (c *Component) waitForDisconnect(ctx context.Context) {
	healthCheck, hasHealthCheck := c.connector.(HealthCheckConnector)
	if hasHealthCheck && c.config.GetHealthCheckInterval() > 0 {
		c.healthCheckLoop(ctx, healthCheck)
	} else {
		<-ctx.Done()
	}
}

func (c *Component) healthCheckLoop(ctx context.Context, hc HealthCheckConnector) {
	ticker := time.NewTicker(c.config.GetHealthCheckInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.config.GetHealthCheckInterval()/2)
			err := hc.SendPing(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Component) waitRetry(attempt int) bool {
	ctx := c.getContext()
	if ctx.Err() != nil {
		return false
	}
	if attempt <= 0 {
		return true
	}
	maxRetries := c.config.GetMaxRetries()
	if maxRetries != -1 && attempt > maxRetries {
		return false
	}
	delay := c.backoff.NextDelay()
	c.eventHandler.OnReconnecting(attempt, delay)
	select {
	case <-ctx.Done():
		// ctx 可能因 tryRefresh 被替换；若已替换则继续后续循环。
		if c.getContext() != ctx {
			return true
		}
		return false
	case <-time.After(delay):
	}
	return true
}

func (c *Component) tryRefresh(currentSeq int64) bool {
	if c.isTerminating() {
		return false
	}
	for {
		seq := atomic.LoadInt64(&c.clientSeq)
		if seq > currentSeq {
			return false
		}
		if atomic.CompareAndSwapInt64(&c.clientSeq, seq, seq+1) {
			c.setConnected(false)
			c.connector.Disconnect()
			c.resetContext()
			return true
		}
	}
}

func (c *Component) setConnected(connected bool) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *Component) getContext() context.Context {
	c.lifecycleMu.RLock()
	defer c.lifecycleMu.RUnlock()
	return c.ctx
}

func (c *Component) cancelContext() {
	c.lifecycleMu.RLock()
	cancel := c.cancel
	c.lifecycleMu.RUnlock()
	cancel()
}

func (c *Component) resetContext() {
	if c.isTerminating() {
		return
	}
	c.lifecycleMu.Lock()
	if c.isTerminating() {
		c.lifecycleMu.Unlock()
		return
	}
	c.cancel()
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.lifecycleMu.Unlock()
}

func (c *Component) isTerminating() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closing || c.stopped
}

func (c *Client) Raw() interface{} {
	return c.raw
}

func (c *Client) Do(fn func(interface{}) error) error {
	for {
		c.comp.mu.Lock()
		if c.comp.clientSeq != c.seq {
			c.seq = c.comp.clientSeq
			if cg, ok := c.comp.connector.(ClientGetter); ok {
				c.raw = cg.GetClient()
			}
		}
		raw := c.raw
		c.comp.mu.Unlock()

		if err := fn(raw); err != nil {
			if c.comp.tryRefresh(c.seq) {
				c.comp.WaitReconnect()
				continue
			}
			return err
		}
		return nil
	}
}
