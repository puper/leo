package config

import "time"

type Config struct {
	Endpoints    []string      `json:"endpoints"`
	DialTimeout  time.Duration `json:"dialTimeout"`
	LeaseTimeout time.Duration `json:"leaseTimeout"`
	InitTimeout  time.Duration `json:"initTimeout"`
	CloseTimeout time.Duration `json:"closeTimeout"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
}
