package reconnect

import (
	"context"
	"errors"
	"sync"
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

	mu          sync.Mutex
	cond        *sync.Cond
	connected   bool
	closing     bool
	stopped     bool
	clientSeq   int64
	started     bool
	raw         interface{}
	cycleCancel context.CancelFunc

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
	if cg, ok := connector.(ClientGetter); ok {
		c.raw = cg.GetClient()
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *Component) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.stopped {
		return errors.New("reconnect component is closed")
	}
	if c.started {
		return nil
	}
	c.started = true
	c.wg.Add(1)
	go c.run()
	return nil
}

func (c *Component) Close() error {
	c.mu.Lock()
	c.closing = true
	c.connected = false
	started := c.started
	c.cond.Broadcast()
	c.mu.Unlock()
	c.cancel()
	if !started {
		return nil
	}
	if timeout := c.config.GetCloseTimeout(); timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-c.doneCh:
		case <-timer.C:
			return context.DeadlineExceeded
		}
	}
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
	return &Client{raw: c.raw, seq: c.clientSeq, comp: c}
}

func (c *Component) WaitReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for !c.connected && !c.stopped && !c.closing {
		c.cond.Wait()
	}
}

func (c *Component) WaitRefresh() {
	c.WaitReconnect()
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
	for c.waitRetry(attempt) {
		ctx, cancel := context.WithCancel(c.ctx)
		c.mu.Lock()
		c.cycleCancel = cancel
		c.mu.Unlock()
		err := c.connector.Connect(ctx)
		if err != nil {
			cancel()
			c.eventHandler.OnError(err)
			attempt++
			continue
		}
		// Connect/Disconnect/GetClient 由同一循环调用，外部只读取已发布的连接。
		var raw interface{}
		if cg, ok := c.connector.(ClientGetter); ok {
			raw = cg.GetClient()
		}
		c.mu.Lock()
		published := !c.closing && ctx.Err() == nil
		if published {
			c.raw = raw
			c.clientSeq++
			c.connected = true
		}
		c.cond.Broadcast()
		c.mu.Unlock()
		if published {
			c.backoff.Reset()
			c.eventHandler.OnConnected()
			c.waitForDisconnect(ctx)
		}
		cancel()
		c.setConnected(false)
		if err := c.connector.Disconnect(); err != nil {
			c.eventHandler.OnError(err)
		}
		if c.ctx.Err() != nil {
			return
		}
		c.eventHandler.OnDisconnected(nil)
		attempt = 1
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
			ctx, cancel := context.WithTimeout(ctx, c.config.GetHealthCheckInterval()/2)
			err := hc.SendPing(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Component) waitRetry(attempt int) bool {
	if c.ctx.Err() != nil {
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
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c *Component) tryRefresh(currentSeq int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.stopped || !c.connected || c.clientSeq != currentSeq {
		return false
	}
	c.connected = false
	c.clientSeq++
	// 只取消这一代连接，断开与下次连接均交给主循环处理。
	c.cycleCancel()
	c.cond.Broadcast()
	return true
}

func (c *Component) setConnected(connected bool) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *Client) Raw() interface{} {
	c.comp.mu.Lock()
	defer c.comp.mu.Unlock()
	return c.raw
}

func (c *Client) Do(fn func(interface{}) error) error {
	for {
		c.comp.WaitReconnect()
		c.comp.mu.Lock()
		if !c.comp.connected {
			c.comp.mu.Unlock()
			return errors.New("reconnect component is not connected")
		}
		if c.comp.clientSeq != c.seq {
			c.seq, c.raw = c.comp.clientSeq, c.comp.raw
		}
		raw, seq := c.raw, c.seq
		c.comp.mu.Unlock()
		if err := fn(raw); err != nil {
			if c.comp.tryRefresh(seq) {
				continue
			}
			return err
		}
		return nil
	}
}
