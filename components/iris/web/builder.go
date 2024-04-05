package web

import (
	"net/http"

	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/iris/web/config"
	"github.com/puper/leo/engine"
)

func Builder(cfg *config.Config, configurers ...func(*Web) error) engine.Builder {
	return func(e *engine.Engine) (any, error) {
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

		go web.app.Run(
			iris.Server(s),
			iris.WithoutServerError(
				iris.ErrServerClosed,
			),
			iris.WithoutPathCorrection,
			iris.WithOptimizations,
		)
		return web, nil
	}
}
