package web

import (
	"context"
	"errors"
	"net"

	"github.com/kataras/iris/v12"
	"github.com/puper/leo/components/iris/web/config"
)

type Web struct {
	config   *config.Config
	app      *iris.Application
	listener net.Listener
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
	shutdownErr := me.app.Shutdown(ctx)
	if me.listener == nil {
		return shutdownErr
	}
	closeErr := me.listener.Close()
	if errors.Is(closeErr, net.ErrClosed) {
		closeErr = nil
	}
	return errors.Join(shutdownErr, closeErr)
}
