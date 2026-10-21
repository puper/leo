package web

import (
	"server/internal/components/gin/web/config"

	"github.com/puper/leo/engine"
)

func Builder(container Container, cfg *config.Config, configureFuncs ...func(*Component) error) engine.Builder {
	return func() (any, error) {
		return New(container, cfg, configureFuncs...)
	}
}
