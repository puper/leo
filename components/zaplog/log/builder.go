package log

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/components/zaplog/log/config"
	"github.com/puper/leo/engine"
)

func Builder(cfg *config.Config, configurers ...func(*Log) error) engine.Builder {
	return func(e *engine.Engine) (any, error) {
		reply, err := New(cfg)
		if err != nil {
			return nil, errors.WithMessage(err, "New")
		}
		for _, configurer := range configurers {
			if err := configurer(reply); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		return reply, err
	}
}
