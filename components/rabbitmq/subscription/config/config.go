package config

import "time"

type Config struct {
	Addr           string        `json:"addr,omitempty"`
	ExchangeName   string        `json:"exchangeName,omitempty"`
	QueueName      string        `json:"queueName,omitempty"`
	RoutingKey     string        `json:"routingKey,omitempty"`
	AutoAck        bool          `json:"autoAck,omitempty"`
	StartTimeout   time.Duration `json:"startTimeout,omitempty"`
	CloseTimeout   time.Duration `json:"closeTimeout,omitempty"`
	ReconnectDelay time.Duration `json:"reconnectDelay,omitempty"`

	ExchangeDeclare bool   `json:"exchangeDeclare,omitempty"`
	ExchangeType    string `json:"exchangeType,omitempty"`
	QueueDeclare    bool   `json:"queueDeclare,omitempty"`
	QueueBind       bool   `json:"queueBind,omitempty"`
	PrefetchCount   int    `json:"prefetchCount,omitempty"`
	PrefetchSize    int    `json:"prefetchSize,omitempty"`
}
