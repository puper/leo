package subscription

import (
	"github.com/pkg/errors"
	"github.com/puper/leo/components/rabbitmq/subscription/config"
	"github.com/puper/leo/engine"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Builder(cfg *config.Config, configurers ...func(*Subscription) error) engine.Builder {
	return func() (any, error) {
		me := New(cfg)
		for _, configurer := range configurers {
			if err := configurer(me); err != nil {
				return nil, errors.WithMessage(err, "configurer")
			}
		}
		if err := me.Start(); err != nil {
			return nil, errors.WithMessage(err, "Start")
		}
		return me, nil
	}
}

func WithSubscriptionCallback(subscriptionCallback func(*amqp.Channel, *config.Config, bool) (<-chan amqp.Delivery, error)) func(*Subscription) error {
	return func(sub *Subscription) error {
		sub.subscriptionCallback = subscriptionCallback
		return nil
	}
}
