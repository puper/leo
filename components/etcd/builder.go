package etcd

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/components/etcd/config"
	"github.com/puper/leo/engine"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (any, error) {
		me, err := clientv3.New(clientv3.Config{
			Endpoints:   cfg.Endpoints,
			DialTimeout: cfg.DialTimeout,
			Username:    cfg.Username,
			Password:    cfg.Password,
		})
		if err != nil {
			return nil, errors.WithMessage(err, "etcd.New")
		}
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "etcd.configurer")
			}
		}
		return me, nil
	}
}
