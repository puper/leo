package log

import (
	stderrors "errors"
	"github.com/pkg/errors"
	"github.com/puper/leo/v2/components/zaplog/log/config"
	"github.com/puper/leo/v2/engine"
)

func Builder(cfg *config.Config, configurers ...func(*Log) error) engine.Builder {
	return func() (any, error) {
		reply, err := New(cfg)
		if err != nil {
			return nil, errors.WithMessage(err, "New")
		}
		for _, configurer := range configurers {
			if err := configurer(reply); err != nil {
				return nil, stderrors.Join(errors.WithMessage(err, "configurer"), reply.Close())
			}
		}
		return reply, err
	}
}
