package localfile

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/v2/engine"
)

func Builder(cfg *Config) engine.Builder {
	return func() (interface{}, error) {
		if cfg == nil {
			return nil, errors.WithMessage(errors.New("config is nil"), "localfile.New")
		}
		return New(cfg)
	}
}
