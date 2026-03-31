package reconnect

import "context"

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
	connected  bool
}

func (c *funcConnector) Connect(ctx context.Context) error {
	if c.connect == nil {
		c.connected = true
		return nil
	}
	err := c.connect(ctx)
	c.connected = err == nil
	return err
}

func (c *funcConnector) Disconnect() error {
	c.connected = false
	if c.disconnect != nil {
		return c.disconnect()
	}
	return nil
}

func (c *funcConnector) IsConnected() bool {
	return c.connected
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
