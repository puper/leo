package server

import (
	"net"

	"github.com/pkg/errors"
	"github.com/puper/leo/components/grpc/server/config"
	"github.com/puper/leo/engine"
	"google.golang.org/grpc"
)

func Builder(cfg *config.Config, configurers ...func(*Component) error) engine.Builder {
	return func() (any, error) {
		me := &Component{
			Server: grpc.NewServer(),
			config: cfg,
		}
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		lis, err := net.Listen("tcp", cfg.Addr)
		if err != nil {
			return nil, errors.WithMessage(err, "net.Listen")
		}
		go func() {
			defer lis.Close()
			if err := me.Server.Serve(lis); err != nil {
				if !errors.Is(err, grpc.ErrServerStopped) {
					// log error?
				}
			}
		}()
		return me, nil
	}
}
