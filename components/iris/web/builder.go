package web

import (
	stderrors "errors"
	"net"
	"net/http"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/iris/web/config"
	"github.com/puper/leo/engine"
)

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
		lis, err := net.Listen("tcp", cfg.Addr)
		if err != nil {
			return nil, errors.WithMessage(err, "net.Listen")
		}

		runErrCh := make(chan error, 1)
		go func() {
			err := web.app.Run(
				iris.Listener(lis),
				iris.Server(s),
				iris.WithoutPathCorrection,
				iris.WithOptimizations,
			)
			if err != nil && !stderrors.Is(err, iris.ErrServerClosed) {
				runErrCh <- err
			}
			close(runErrCh)
		}()
		select {
		case err := <-runErrCh:
			if err != nil {
				return nil, errors.WithMessage(err, "app.Run")
			}
		case <-time.After(50 * time.Millisecond):
		}
		return web, nil
	}
}
