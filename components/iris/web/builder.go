package web

import (
	"context"
	stderrors "errors"
	"net"
	"net/http"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/iris/web/config"
	"github.com/puper/leo/engine"
)

const defaultStartCheckTimeout = 300 * time.Millisecond

func Builder(cfg *config.Config, configurers ...func(*Web) error) engine.Builder {
	return func() (any, error) {
		s := &http.Server{
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			Addr:         cfg.Addr,
		}
		web := &Web{
			config: cfg,
			app:    iris.New(),
		}
		for _, configurer := range configurers {
			if err := configurer(web); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		web.app.Configure(iris.WithoutPathCorrection, iris.WithOptimizations)
		if err := web.app.Build(); err != nil {
			return nil, errors.WithMessage(err, "app.Build")
		}
		lis, err := net.Listen("tcp", cfg.Addr)
		if err != nil {
			return nil, errors.WithMessage(err, "net.Listen")
		}

		web.listener = lis
		runErrCh := make(chan error, 1)
		ready := make(chan struct{})
		go func() {
			err := web.app.Run(
				func(app *iris.Application) error {
					host := app.NewHost(s)
					close(ready)
					return host.Serve(lis)
				},
			)
			if err != nil {
				lis.Close()
				if !stderrors.Is(err, iris.ErrServerClosed) {
					runErrCh <- err
				}
			}
		}()
		timer := time.NewTimer(getStartCheckTimeout(cfg))
		defer timer.Stop()
		select {
		case <-ready:
			return web, nil
		case err := <-runErrCh:
			s.Close()
			return nil, errors.WithMessage(err, "app.Run")
		case <-timer.C:
			lis.Close()
			s.Close()
			return nil, errors.WithMessage(context.DeadlineExceeded, "app.Run startup")
		}
	}
}

func getStartCheckTimeout(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.StartCheckTimeout > 0 {
		return cfg.StartCheckTimeout
	}
	return defaultStartCheckTimeout
}
