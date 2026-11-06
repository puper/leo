package reconnect

import (
	"context"
	"sync/atomic"
)

type Connector interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
}

type ConfigProvider interface {
	Config() ReconnectConfig
}

type HealthCheckConnector interface {
	Connector
	SendPing(ctx context.Context) error
}

type ClientGetter interface {
	GetClient() interface{}
}

type ConnectFunc func(ctx context.Context) error
type DisconnectFunc func() error

func AsConnector(conn ConnectFunc, disconnect DisconnectFunc) Connector {
	return &funcConnector{
		connect:    conn,
		disconnect: disconnect,
	}
}

type funcConnector struct {
	connect    ConnectFunc
	disconnect DisconnectFunc
	connected  atomic.Bool
}

func (c *funcConnector) Connect(ctx context.Context) error {
	if c.connect == nil {
		c.connected.Store(true)
		return nil
	}
	err := c.connect(ctx)
	c.connected.Store(err == nil)
	return err
}

func (c *funcConnector) Disconnect() error {
	c.connected.Store(false)
	if c.disconnect != nil {
		return c.disconnect()
	}
	return nil
}

func (c *funcConnector) IsConnected() bool {
	return c.connected.Load()
}

func AsConfigProvider(cfg ReconnectConfig) ConfigProvider {
	return &configProvider{cfg: cfg}
}

type configProvider struct {
	cfg ReconnectConfig
}

func (c *configProvider) Config() ReconnectConfig {
	return c.cfg
}
