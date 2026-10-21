package uniqid

import (
	stderrors "errors"

	"github.com/pkg/errors"
	"github.com/puper/leo/v2/components/uniqid/config"
	"github.com/puper/leo/v2/engine"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (interface{}, error) {
		me := New(cfg)
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, stderrors.Join(errors.WithMessage(err, "uniqid.configurer"), me.Close())
			}
		}
		if me.etcdCli == nil {
			return nil, stderrors.Join(errors.WithMessage(errors.New("etcd client is nil"), "uniqid.configurer"), me.Close())
		}
		if err := me.Start(); err != nil {
			return nil, errors.WithMessage(err, "uniqid.Start")
		}
		return me, nil
	}
}

func WithEtcd(f func() *clientv3.Client) func(*Component) error {
	return func(me *Component) error {
		me.etcdCli = f()
		return nil
	}
}
