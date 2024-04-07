package config

import "time"

type Config struct {
	ServerUrl           string        `json:"serverUrl,omitempty"`
	Token               string        `json:"token,omitempty"`
	Org                 string        `json:"org,omitempty"`
	Bucket              string        `json:"bucket,omitempty"`
	DailTimeout         time.Duration `json:"dailTimeout,omitempty"`
	TLSHandshakeTimeout time.Duration `json:"tlsHandshakeTimeout,omitempty"`
	InsecureSkipVerify  bool          `json:"insecureSkipVerify,omitempty"`
	MaxIdleConns        int           `json:"maxIdleConns,omitempty"`
	MaxIdleConnsPerHost int           `json:"maxIdleConnsPerHost,omitempty"`
	IdleConnTimeout     time.Duration `json:"idleConnTimeout,omitempty"`
	UseGzip             bool          `json:"useGzip,omitempty"`

	AppName string `json:"appName,omitempty"`
}
