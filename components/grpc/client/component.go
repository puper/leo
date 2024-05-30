package client

import (
	"github.com/puper/leo/components/grpc/client/config"
	"google.golang.org/grpc"
)

type Component struct {
	*grpc.ClientConn
	config *config.Config
}

func (me *Component) Close() error {
	if me.ClientConn != nil {
		return me.ClientConn.Close()
	}
	return nil
}
