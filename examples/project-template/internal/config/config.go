package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	webConf "server/internal/components/gin/web/config"

	dbConf "github.com/puper/leo/v2/components/db/config"
	logConf "github.com/puper/leo/v2/components/zaplog/log/config"
	"github.com/spf13/viper"
)

var instance atomic.Pointer[Config]

func Instance() *Config       { return instance.Load() }
func SetInstance(cfg *Config) { instance.Store(cfg) }

type Config struct {
	App *App            `json:"app"`
	Log *logConf.Config `json:"log"`
	Web *webConf.Config `json:"web"`
	Db  *dbConf.Config  `json:"db"`
}

type App struct {
	Env      string      `json:"env"`
	TimeZone string      `json:"timezone"`
	NodeID   *int64      `json:"nodeID"`
	Auth     *AuthConfig `json:"auth"`
}

type AuthConfig struct {
	JwtSecret string `json:"jwtSecret"`
	JwtExpire int    `json:"jwtExpire"`
}

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

func (cfg *AuthConfig) SigningKey() ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("app.auth is required")
	}
	key, err := base64.URLEncoding.DecodeString(cfg.JwtSecret)
	if err != nil || len(key) < 32 {
		return nil, errors.New("app.auth.jwtSecret must be URL-safe base64 encoding of at least 32 random bytes")
	}
	if cfg.JwtExpire <= 0 {
		return nil, errors.New("app.auth.jwtExpire must be positive")
	}
	return key, nil
}

// Load 仅在启动时加载，避免配置热更新与已构建的组件状态不一致。
func Load(v *viper.Viper) error {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return errors.New("invalid configuration structure")
	}
	if v.IsSet("db") && cfg.Db == nil {
		return errors.New("db configuration must define servers.server; omit db to run without a database")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	instance.Store(&cfg)
	return nil
}

func (cfg *Config) Validate() error {
	if cfg.App == nil {
		return errors.New("app is required")
	}
	if cfg.App.Env != EnvDev && cfg.App.Env != EnvProd {
		return errors.New("app.env must be dev or prod")
	}
	if cfg.App.TimeZone == "" {
		return errors.New("app.timezone is required")
	}
	if _, err := time.LoadLocation(cfg.App.TimeZone); err != nil {
		return errors.New("invalid app.timezone")
	}
	if cfg.App.NodeID == nil || *cfg.App.NodeID < 0 || *cfg.App.NodeID > 1023 {
		return errors.New("app.nodeID must be explicitly set between 0 and 1023")
	}
	if _, err := cfg.App.Auth.SigningKey(); err != nil {
		return err
	}
	if cfg.Log == nil || cfg.Log.Logs["default"] == nil {
		return errors.New("log.logs.default is required")
	}
	for name, log := range cfg.Log.Logs {
		if log == nil {
			return fmt.Errorf("log.logs.%s is required", name)
		}
	}
	if cfg.Web == nil || strings.TrimSpace(cfg.Web.Addr) == "" {
		return errors.New("web.addr is required")
	}
	if err := cfg.Web.Validate(); err != nil {
		return err
	}
	if cfg.Db == nil {
		return nil
	}
	if _, ok := cfg.Db.Servers["server"]; !ok {
		return errors.New("db.servers.server is required")
	}
	for name, db := range cfg.Db.Servers {
		if db.Driver != "mysql" || strings.TrimSpace(db.Master) == "" || db.Migrate == nil {
			return fmt.Errorf("db.servers.%s requires mysql driver, master and migrate configuration", name)
		}
	}
	return nil
}
