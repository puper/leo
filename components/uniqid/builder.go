package uniqid

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/components/uniqid/config"
	"github.com/puper/leo/engine"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (interface{}, error) {
		me := New(cfg)
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "uniqid.configurer")
			}
		}
		if me.etcdCli == nil {
			return nil, errors.WithMessage(errors.New("etcd client is nil"), "uniqid.configurer")
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

func WithSubscriptions(chs ...chan *Event) func(*Component) error {
	return func(me *Component) error {
		me.chs = chs
		return nil
	}
}
