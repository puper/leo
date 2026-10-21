package logging

import (
	"server/internal/ifaces/logger"

	"github.com/puper/leo/v2/engine"
)

type Container interface {
	logger.GetLogger
}

func Builder(container Container) engine.Builder {
	return func() (any, error) {
		SetLogger(container.GetLogger())
		return struct{}{}, nil
	}
}
