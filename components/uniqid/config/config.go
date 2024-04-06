package config

import "time"

type Config struct {
	LeaseTimeout time.Duration `json:"leaseTimeout"`
	InitTimeout  time.Duration `json:"initTimeout"`
	CloseTimeout time.Duration `json:"closeTimeout"`
	KeyPrefix    string        `json:"keyPrefix"`
	MinId        int           `json:"minId"`
	MaxId        int           `json:"maxId"`
}
