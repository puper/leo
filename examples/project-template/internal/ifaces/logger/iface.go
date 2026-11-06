package logger

import "go.uber.org/zap"

type GetLogger interface {
	GetLogger(...string) *zap.SugaredLogger
}
