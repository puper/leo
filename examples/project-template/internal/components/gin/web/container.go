package web

import "go.uber.org/zap"

type Container interface {
	GetLogger(...string) *zap.SugaredLogger
}
