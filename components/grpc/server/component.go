package server

import (
	"context"
	"github.com/puper/leo/components/grpc/server/config"
	"google.golang.org/grpc"
	"time"
)

type Component struct {
	*grpc.Server
	config *config.Config
}

func (me *Component) Close() error {
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
