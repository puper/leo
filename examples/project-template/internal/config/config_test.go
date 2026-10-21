package config

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	original := Instance()
	t.Cleanup(func() { SetInstance(original) })
	load := func() *viper.Viper {
		v := viper.New()
		v.SetConfigFile("../../config/config.example.toml")
		if err := v.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			t.Fatal(err)
		}
		v.Set("app.auth.jwtSecret", base64.URLEncoding.EncodeToString(secret))
		return v
	}
	if err := Load(load()); err != nil {
		t.Fatal(err)
	}
	valid := Instance()
	if valid.Db != nil {
		t.Fatal("default configuration unexpectedly enables database")
	}
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"app.auth.jwtSecret", ""}, {"app.auth.jwtExpire", 0},
		{"app.env", "invalid"}, {"app.timezone", "invalid"},
		{"app.nodeID", -1}, {"app.nodeID", 1024},
		{"db", map[string]any{}},
		{"db.servers.server.master", ""}, {"web.corsOrigins", []string{"*"}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			v := load()
			v.Set(tc.key, tc.value)
			if err := Load(v); err == nil {
				t.Fatal("invalid config accepted")
			}
			if Instance() != valid {
				t.Fatal("invalid config replaced valid instance")
			}
		})
	}
}

func TestOptionalMySQLConfig(t *testing.T) {
	original := Instance()
	t.Cleanup(func() { SetInstance(original) })
	load := func() *viper.Viper {
		v := viper.New()
		v.SetConfigFile("../../config/config.example.toml")
		if err := v.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			t.Fatal(err)
		}
		v.Set("app.auth.jwtSecret", base64.URLEncoding.EncodeToString(secret))
		v.Set("db.servers.server.driver", "mysql")
		v.Set("db.servers.server.master", "user@tcp(localhost:3306)/example")
		v.Set("db.servers.server.migrate.tableName", "schema_migrations")
		return v
	}
	if err := Load(load()); err != nil {
		t.Fatal(err)
	}
	if Instance().Db == nil {
		t.Fatal("explicit database configuration was ignored")
	}
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"db.servers.server.driver", "postgres"},
		{"db.servers.server.master", ""},
		{"db.servers.server", map[string]any{"driver": "mysql", "master": "user@tcp(localhost:3306)/example"}},
	} {
		v := load()
		v.Set(tc.key, tc.value)
		if err := Load(v); err == nil {
			t.Errorf("invalid explicit database configuration accepted: %s", tc.key)
		}
	}
}
