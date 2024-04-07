package db

import (
	"embed"

	"github.com/pkg/errors"
	"github.com/puper/leo/components/db/config"
	"github.com/puper/leo/engine"
	"gorm.io/gorm"
)

func Builder(cfg *config.Config, configurers ...func(*Db) error) engine.Builder {
	return func() (any, error) {
		for k, v := range cfg.Servers {
			cfg.Servers[k] = v
		}
		reply, err := New(cfg)
		if err != nil {
			return nil, errors.WithMessage(err, "New")
		}
		for _, configurer := range configurers {
			if err := configurer(reply); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		return reply, nil
	}
}

func WithConnCallback(f func(db *gorm.DB)) func(*Db) error {
	return func(me *Db) error {
		for _, w := range me.wrappers {
			f(w.master)
			for _, s := range w.slave {
				f(s)
			}
		}
		return nil
	}
}

func WithMigrateFs(migrateFs embed.FS) func(*Db) error {
	return func(me *Db) error {
		migrates, err := LoadMigrates(migrateFs)
		if err != nil {
			return errors.WithMessage(err, "LoadMigrates")
		}
		for name, w := range me.wrappers {
			cfg, ok := me.config.Servers[name]
			if !ok {
				continue
			}
			if ms, ok := migrates[name]; ok {
				m := NewMigrate(w.Write(), cfg.Migrate, ms)
				if err := m.Migrate(); err != nil {
					if err != ErrNoMigrationDefined {
						return errors.WithMessagef(err, "migrate %s", name)
					}
				}
			}
		}
		return nil
	}
}
