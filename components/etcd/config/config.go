package config

import "time"

type Config struct {
	Endpoints   []string      `json:"endpoints"`
	DialTimeout time.Duration `json:"dialTimeout"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
}
