package reconnect

import "time"

type ReconnectConfig interface {
	GetMaxRetries() int
	GetInitialInterval() time.Duration
	GetMaxInterval() time.Duration
	GetMultiplier() float64
	GetCloseTimeout() time.Duration
	GetHealthCheckInterval() time.Duration
}

type DefaultReconnectConfig struct {
	MaxRetries          int
	InitialInterval     time.Duration
	MaxInterval         time.Duration
	Multiplier          float64
	CloseTimeout        time.Duration
	HealthCheckInterval time.Duration
}

func (c *DefaultReconnectConfig) GetMaxRetries() int {
	if c.MaxRetries == 0 {
		return -1
	}
	return c.MaxRetries
}

func (c *DefaultReconnectConfig) GetInitialInterval() time.Duration {
	if c.InitialInterval == 0 {
		return time.Second
	}
	return c.InitialInterval
}

func (c *DefaultReconnectConfig) GetMaxInterval() time.Duration {
	if c.MaxInterval == 0 {
		return 30 * time.Second
	}
	return c.MaxInterval
}

func (c *DefaultReconnectConfig) GetMultiplier() float64 {
	if c.Multiplier == 0 {
		return 2.0
	}
	return c.Multiplier
}

func (c *DefaultReconnectConfig) GetCloseTimeout() time.Duration {
	if c.CloseTimeout == 0 {
		return 10 * time.Second
	}
	return c.CloseTimeout
}

func (c *DefaultReconnectConfig) GetHealthCheckInterval() time.Duration {
	return c.HealthCheckInterval
}
