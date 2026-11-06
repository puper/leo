package config

import (
	"errors"
	"net/url"
	"strings"
)

type Config struct {
	Addr               string
	ReadHeaderTimeout  int
	ReadTimeout        int
	WriteTimeout       int
	MaxHeaderBytes     int
	MaxMultipartMemory int64
	CorsOrigins        []string
}

func (cfg *Config) Validate() error {
	if cfg.ReadHeaderTimeout < 0 || cfg.ReadTimeout < 0 || cfg.WriteTimeout < 0 || cfg.MaxHeaderBytes < 0 || cfg.MaxMultipartMemory < 0 {
		return errors.New("web timeouts and size limits must not be negative")
	}
	for _, origin := range cfg.CorsOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(origin, "*") {
			return errors.New("web.corsOrigins must contain explicit HTTP(S) origins without paths or wildcards")
		}
	}
	return nil
}
