package config

import "time"

type Config struct {
	ReadTimeout     time.Duration `json:"readTimeout"`
	WriteTimeout    time.Duration `json:"writeTimeout"`
	IdleTimeout     time.Duration `json:"idleTimeout"`
	ShutdownTimeout time.Duration `json:"shutdownTimeout"`
	Addr            string        `json:"addr"`
}
