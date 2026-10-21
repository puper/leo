package bootstrap

import (
	"context"
	stderrors "errors"
	"net/http"
	"time"
	_ "time/tzdata"

	"server/internal/app"
	"server/internal/components"
	"server/internal/components/gin/web"
	"server/internal/components/idgen"
	"server/internal/config"
	"server/internal/logging"
	"server/internal/modules/example"
	"server/internal/route"
	_ "server/internal/swagger/docs"
	"server/migrates"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/db"
	"github.com/puper/leo/components/zaplog/log"
	"github.com/puper/leo/engine"
	"github.com/spf13/viper"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Bootstrap(cfgFile string) (err error) {
	cfg := viper.New()
	cfg.SetConfigFile(cfgFile)
	if err := cfg.ReadInConfig(); err != nil {
		return errors.New("cannot read configuration file; check path and TOML syntax")
	}
	if err := config.Load(cfg); err != nil {
		return err
	}
	conf := config.Instance()
	time.Local, err = time.LoadLocation(conf.App.TimeZone)
	if err != nil {
		return err
	}
	if conf.App.Env == config.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	e := engine.New(cfg)
	app.Set(e)
	defer func() { err = stderrors.Join(err, e.Close()) }()
	container := app.GetContainer()
	e.Register(components.Log, log.Builder(conf.Log))
	e.Register(components.Logging, logging.Builder(container), components.Log)
	e.Register(components.IdGen, idgen.Builder(*conf.App.NodeID))
	webDependencies := []string{components.Logging}
	if conf.Db != nil {
		e.Register(components.Db, db.Builder(conf.Db, db.WithMigrateFs(migrates.Fs)), components.Logging)
		e.Register(components.Example, example.Builder(container), components.Db, components.IdGen)
		webDependencies = append(webDependencies, components.Example)
	}
	e.Register(components.Web, web.Builder(container, conf.Web, func(w *web.Component) error {
		r := w.GetGinEngine()
		r.GET("/readiness", func(c *gin.Context) {
			if conf.Db != nil {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				defer cancel()
				sqlDB, err := container.GetDb().Write("server").DB()
				if err != nil || sqlDB.PingContext(ctx) != nil {
					c.String(http.StatusServiceUnavailable, "not ready")
					return
				}
			}
			c.String(http.StatusOK, "ok")
		})
		if conf.App.Env == config.EnvDev {
			r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
		}
		for _, register := range route.GetRouteFuncs() {
			if err := register(r); err != nil {
				return err
			}
		}
		return nil
	}), webDependencies...)
	if err := e.Build(); err != nil {
		return errors.WithMessage(err, "build engine")
	}
	return e.Wait()
}
