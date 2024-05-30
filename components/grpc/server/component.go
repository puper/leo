package server

import (
	"github.com/puper/leo/components/grpc/server/config"
	"google.golang.org/grpc"
)

type Component struct {
	config *config.Config
	server *grpc.Server
}

func (me *Component) Close() error {
	if me.server != nil {
		me.server.GracefulStop()
	}
	return nil
}
