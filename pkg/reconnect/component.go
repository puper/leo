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

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu        sync.Mutex
	cond      *sync.Cond
	connected bool
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
	c.wg.Add(1)
	go c.run()
	return nil
}

func (c *Component) Close() error {
	c.cancel()
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
	for !c.connected {
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
	defer close(c.doneCh)
	defer c.backoff.Reset()

	c.connectLoop()
}

func (c *Component) connectLoop() {
	for {
		err := c.connector.Connect(c.ctx)
		if err == nil {
			c.mu.Lock()
			c.connected = true
			c.mu.Unlock()
			c.eventHandler.OnConnected()
			c.cond.Broadcast()

			c.waitForDisconnect()

			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()

			if c.ctx.Err() != nil {
				c.connector.Disconnect()
				return
			}
			c.eventHandler.OnDisconnected(nil)
		} else {
			c.eventHandler.OnError(err)
		}

		if !c.reconnect() {
			return
		}
	}
}

func (c *Component) waitForDisconnect() {
	healthCheck, hasHealthCheck := c.connector.(HealthCheckConnector)
	if hasHealthCheck && c.config.GetHealthCheckInterval() > 0 {
		c.healthCheckLoop(healthCheck)
	} else {
		<-c.ctx.Done()
	}
}

func (c *Component) healthCheckLoop(hc HealthCheckConnector) {
	ticker := time.NewTicker(c.config.GetHealthCheckInterval())
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
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

func (c *Component) reconnect() bool {
	maxRetries := c.config.GetMaxRetries()
	for attempt := 1; maxRetries == -1 || attempt <= maxRetries; attempt++ {
		delay := c.backoff.NextDelay()
		c.eventHandler.OnReconnecting(attempt, delay)
		select {
		case <-c.ctx.Done():
			return false
		case <-time.After(delay):
		}
		err := c.connector.Connect(c.ctx)
		if err == nil {
			return true
		}
		c.eventHandler.OnError(err)
	}
	return false
}

func (c *Component) tryRefresh(currentSeq int64) bool {
	for {
		seq := atomic.LoadInt64(&c.clientSeq)
		if seq > currentSeq {
			return false
		}
		if atomic.CompareAndSwapInt64(&c.clientSeq, seq, seq+1) {
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()
			c.connector.Disconnect()
			c.cancel()
			c.ctx, c.cancel = context.WithCancel(context.Background())
			return true
		}
	}
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
