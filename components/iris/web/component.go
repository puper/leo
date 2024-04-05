package web

import (
	"context"

	"github.com/kataras/iris/v12"
	"github.com/puper/leo/components/iris/web/config"
)

type Web struct {
	config *config.Config
	app    *iris.Application
}

func (me *Web) GetApp() *iris.Application {
	return me.app
}

func (me *Web) Close() error {
	if me.config.ShutdownTimeout > 0 {
		return me.app.Shutdown(context.Background())
	}
	return me.app.Shutdown(context.Background())
}
