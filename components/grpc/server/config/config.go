package config

import "time"

type Config struct {
	Addr            string        `json:"addr"`
	ShutdownTimeout time.Duration `json:"shutdownTimeout"`
}
