package server

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/puper/leo/v2/components/grpc/server/config"
	"google.golang.org/grpc"
)

type Component struct {
	*grpc.Server
	config   *config.Config
	listener net.Listener
}

func (me *Component) Close() (err error) {
	defer func() {
		if me.listener == nil {
			return
		}
		// Serve 可能尚未登记监听器，Close 不能依赖它后续获得调度才释放端口。
		closeErr := me.listener.Close()
		if !errors.Is(closeErr, net.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}()
	if me.Server == nil {
		return nil
	}
	if me.config == nil || me.config.ShutdownTimeout <= 0 {
		me.Server.GracefulStop()
		return nil
	}
	done := make(chan struct{})
	go func() { me.Server.GracefulStop(); close(done) }()
	timer := time.NewTimer(me.config.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		go me.Server.Stop()
		return context.DeadlineExceeded
	}
}
