package db

import (
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/puper/leo/components/db/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type (
	Db struct {
		config   *config.Config
		wrappers map[string]*Wrapper
	}
	Wrapper struct {
		master *gorm.DB
		slave  []*gorm.DB
	}
	Model interface {
		ConnectionName() string
	}
)

func New(cfg *config.Config) (*Db, error) {
	rand.Int()
	man := &Db{
		config:   cfg,
		wrappers: make(map[string]*Wrapper),
	}
	var err error
	for name, config := range cfg.Servers {
		w := new(Wrapper)
		w.master, err = gorm.Open(mysql.Open(config.Master))
		if err != nil {
			return nil, errors.WithMessage(err, "gorm.Open")
		}
		stdDb, err := w.master.DB()
		if err != nil {
			return nil, errors.WithMessage(err, "master.DB")
		}
		stdDb.SetConnMaxLifetime(time.Duration(config.ConnMaxLifeTime) * time.Second)
		stdDb.SetMaxIdleConns(config.MaxIdleConns)
		stdDb.SetMaxOpenConns(config.MaxOpenConns)
		for _, s := range config.Slave {
			slave, err := gorm.Open(mysql.Open(s))
			if err != nil {
				return nil, err
			}
			stdDb, err := slave.DB()
			if err != nil {
				return nil, errors.WithMessage(err, "slave.DB")
			}
			stdDb.SetConnMaxLifetime(time.Duration(config.ConnMaxLifeTime) * time.Second)
			stdDb.SetMaxIdleConns(config.MaxIdleConns)
			stdDb.SetMaxOpenConns(config.MaxOpenConns)
			w.slave = append(w.slave, slave)
		}
		man.wrappers[name] = w
	}
	return man, nil
}

func (me *Wrapper) Write() *gorm.DB {
	return me.master
}

func (me *Wrapper) Read() *gorm.DB {
	if len(me.slave) == 0 {
		return me.master
	}
	return me.slave[rand.Intn(len(me.slave))]
}

func (me *Db) Write(name string) *gorm.DB {
	return me.wrappers[name].Write()
}

func (me *Db) Read(name string) *gorm.DB {
	return me.wrappers[name].Read()
}

func (me *Db) WriteModel(m Model) *gorm.DB {
	return me.Write(m.ConnectionName()).Model(m)
}

func (me *Db) ReadModel(m Model) *gorm.DB {
	return me.Read(m.ConnectionName()).Model(m)
}

func (me *Db) Close() error {
	var errs []error
	for _, w := range me.wrappers {
		if db, err := w.master.DB(); err == nil {
			errs = append(errs, db.Close())
		}
		for _, s := range w.slave {
			if db, err := s.DB(); err == nil {
				errs = append(errs, db.Close())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New("db close errors")
	}
	return nil
}
