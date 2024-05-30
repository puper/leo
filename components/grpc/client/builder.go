package client

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/components/grpc/client/config"
	"github.com/puper/leo/engine"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (any, error) {
		me := &Component{
			config: cfg,
		}
		cli, err := grpc.NewClient(
			cfg.Addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, errors.WithMessage(err, "")
		}
		me.client = cli
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		return me, nil
	}
}
