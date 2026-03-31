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
	if me == nil || me.app == nil {
		return nil
	}
	ctx := context.Background()
	if me.config != nil && me.config.ShutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, me.config.ShutdownTimeout)
		defer cancel()
	}
	return me.app.Shutdown(ctx)
}
