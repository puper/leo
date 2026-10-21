package log

import (
	"errors"
	"io"
	"os"

	"github.com/puper/leo/components/zaplog/log/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Log struct {
	config  *config.Config
	logs    map[string]*zap.SugaredLogger
	writers []io.Closer
}

func New(cfg *config.Config) (*Log, error) {
	instance := &Log{
		config: cfg,
		logs:   make(map[string]*zap.SugaredLogger),
	}
	for logName, logCfg := range cfg.Logs {
		logger, writer := newLog(logCfg)
		instance.logs[logName] = logger
		if writer != nil {
			instance.writers = append(instance.writers, writer)
		}
	}
	if _, ok := instance.logs["default"]; !ok {
		instance.logs["default"] = NewLog(&config.LogConfig{})
	}
	return instance, nil
}

func NewLog(logCfg *config.LogConfig) *zap.SugaredLogger {
	logger, _ := newLog(logCfg)
	return logger
}

func newLog(logCfg *config.LogConfig) (*zap.SugaredLogger, io.Closer) {
	levelMap := map[string]zapcore.Level{
		"debug":  zapcore.DebugLevel,
		"info":   zapcore.InfoLevel,
		"warn":   zapcore.WarnLevel,
		"error":  zapcore.ErrorLevel,
		"dpanic": zapcore.DPanicLevel,
		"panic":  zapcore.PanicLevel,
		"fatal":  zapcore.FatalLevel,
	}
	cfg := zap.NewProductionConfig()
	if lvl, ok := levelMap[logCfg.Level]; ok {
		cfg.Level = zap.NewAtomicLevelAt(lvl)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(levelMap["info"])
	}
	cfg.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	var enc zapcore.Encoder
	if logCfg.Format == "json" {
		enc = zapcore.NewJSONEncoder(cfg.EncoderConfig)
	} else {
		enc = zapcore.NewConsoleEncoder(cfg.EncoderConfig)
	}
	var w zapcore.WriteSyncer
	var closer io.Closer
	if logCfg.Output == "" {
		w = zapcore.AddSync(os.Stdout)
	} else {
		writer := &lumberjack.Logger{
			Filename:   logCfg.Output,
			MaxSize:    logCfg.MaxSize, // megabytes
			MaxAge:     logCfg.MaxAge,  // days
			MaxBackups: logCfg.MaxBackups,
			LocalTime:  true,
			Compress:   logCfg.Compress,
		}
		closer = writer
		w = zapcore.AddSync(writer)
	}
	log := zap.New(
		zapcore.NewCore(enc, w, cfg.Level),
	)
	opts := []zap.Option{}
	opts = append(opts, zap.AddCaller())
	if lvl, ok := levelMap[logCfg.TraceLevel]; ok {
		opts = append(opts, zap.AddStacktrace(lvl))
	} else {
		opts = append(opts, zap.AddStacktrace(zap.ErrorLevel))
	}
	log = log.WithOptions(opts...)
	return log.Sugar().With(logCfg.InitialFields...), closer
}

func (me *Log) Get(names ...string) *zap.SugaredLogger {
	name := "default"
	if len(names) > 0 {
		name = names[0]
	}
	l, ok := me.logs[name]
	if ok {
		return l
	}
	return me.logs["default"]
}

func (me *Log) Close() error {
	// lumberjack 的 Write 不缓冲；zapcore.AddSync 的空 Sync 不会关闭它。
	// 仅释放本组件拥有的文件 writer，不关闭进程共享的 stdout。
	var errs []error
	for _, writer := range me.writers {
		errs = append(errs, writer.Close())
	}
	return errors.Join(errs...)
}
