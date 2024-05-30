package server

import (
	"github.com/puper/leo/components/grpc/server/config"
	"google.golang.org/grpc"
)

type Component struct {
	*grpc.Server
	config *config.Config
}

func (me *Component) Close() error {
	if me.Server != nil {
		me.Server.GracefulStop()
	}
	return nil
}
