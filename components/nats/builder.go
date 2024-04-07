package nats

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/puper/leo/components/nats/config"
	"github.com/puper/leo/engine"
)

func Builder(cfg *config.Config, configurers ...func(*nats.Conn) error) engine.Builder {
	return func() (any, error) {
		c, err := nats.Connect(
			cfg.Url,
			nats.UserInfo(cfg.Username, cfg.Password),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(time.Second*3),
			nats.ErrorHandler(func(c *nats.Conn, sub *nats.Subscription, err error) {
				log.Println("conn", c, "sub", sub, "error", err)
			}),
		)
		if err != nil {
			return nil, errors.WithMessage(err, "connect nats")
		}
		for _, configurer := range configurers {
			if err := configurer(c); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		return c, nil
	}
}
