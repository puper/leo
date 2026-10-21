package logging

import "go.uber.org/zap"

var globalLogger = zap.NewNop().Sugar()

func SetLogger(logger *zap.SugaredLogger) {
	if logger == nil {
		globalLogger = zap.NewNop().Sugar()
		return
	}
	globalLogger = logger
}

func GetLogger() *zap.SugaredLogger {
	return globalLogger
}
