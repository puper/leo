package client

import (
	"github.com/puper/leo/components/grpc/client/config"
	"google.golang.org/grpc"
)

type Component struct {
	config *config.Config
	client *grpc.ClientConn
}

func (me *Component) Close() error {
	if me.client != nil {
		return me.client.Close()
	}
	return nil
}
