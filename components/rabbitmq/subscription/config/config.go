package config

import "time"

type Config struct {
	Addr           string        `json:"addr,omitempty"`
	ExchangeName   string        `json:"exchangeName,omitempty"`
	QueueName      string        `json:"queueName,omitempty"`
	AutoAck        bool          `json:"autoAck,omitempty"`
	CloseTimeout   time.Duration `json:"closeTimeout,omitempty"`
	ReconnectDelay time.Duration `json:"reconnectDelay,omitempty"`
}
