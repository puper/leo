package subscription

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/puper/gcache"
	"github.com/puper/leo/components/rabbitmq/subscription/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

func New(cfg *config.Config) *Subscription {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscription{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		inited: false,
		initCh: make(chan error, 1),
		msgCh:  make(chan *Message, 1),

		cache: gcache.New(0).Build(),
	}
}

type Subscription struct {
	config *config.Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	inited bool
	initCh chan error

	subscriptionCallback func(*amqp.Channel, *config.Config, bool) (<-chan amqp.Delivery, error)

	msgCh chan *Message
	cache gcache.Cache
}

type SubscriptionConfig struct {
	Addr         string
	ExchangeName string
	QueueName    string

	CloseTimeout   time.Duration
	ReconnectDelay time.Duration
}

func (me *Subscription) Start() error {
	if me.subscriptionCallback == nil {
		me.subscriptionCallback = func(ch *amqp.Channel, cfg *config.Config, inited bool) (<-chan amqp.Delivery, error) {
			return ch.Consume(
				cfg.QueueName,
				"",                // Consumer
				me.config.AutoAck, // Auto-Ack
				false,             // Exclusive
				false,             // No-local
				false,             // No-Wait
				nil,               // Args
			)
		}
	}
	me.wg.Add(1)
	go me.mainloop()
	return <-me.initCh

}

func (me *Subscription) MsgCh() <-chan *Message {
	return me.msgCh
}

func (me *Subscription) Close() error {
	me.cancel()
	me.wg.Wait()
	return me.ctx.Err()
}

// 只能在onConnected中调用
func (me *Subscription) ClearMsgs() {
	for {
		if len(me.msgCh) > 0 {
			<-me.msgCh
		} else {
			break
		}
	}
}
func (me *Subscription) mainloop() {
	defer me.wg.Done()
	defer me.cancel()
	for {
		signalCh := make(chan struct{}, 1)
		doneCh := make(chan struct{}, 1)
		me.run(signalCh, doneCh)
		// log error?
		select {
		case <-me.ctx.Done():
			close(signalCh)
			select {
			case <-doneCh:
			case <-time.After(me.config.CloseTimeout):
			}
			return
		case <-doneCh:
			select {
			case <-me.ctx.Done():
				return
			case <-time.After(me.config.ReconnectDelay):
			}
		}
	}
}

func (me *Subscription) run(signalCh chan struct{}, doneCh chan struct{}) error {
	defer close(doneCh)
	conn, err := amqp.Dial(me.config.Addr)
	if err != nil {
		if !me.inited {
			me.initCh <- err
		}
		return err
	}
	defer conn.Close()
	connCloseCh := make(chan *amqp.Error, 1)
	conn.NotifyClose(connCloseCh)
	ch, err := conn.Channel()
	if err != nil {
		if !me.inited {
			me.initCh <- err
		}
		return err
	}
	defer ch.Close()
	chClosedCh := make(chan *amqp.Error, 1)
	ch.NotifyClose(chClosedCh)
	chCancelCh := make(chan string, 1)
	ch.NotifyCancel(chCancelCh)
	deliveries, err := me.subscriptionCallback(ch, me.config, me.inited)
	if err != nil {
		if !me.inited {
			me.initCh <- err
		}
		return err
	}
	if !me.inited {
		me.inited = true
		me.initCh <- nil
	}
	for {
		select {
		case <-signalCh:
			return nil
		case err := <-connCloseCh:
			return err
		case err := <-chClosedCh:
			return err
		case c := <-chCancelCh:
			return fmt.Errorf("cancel: %s", c)
		case d := <-deliveries:
			msg := &Message{
				Delivery: d,
				config:   me.config,
				cache:    me.cache,
			}
			if !msg.IsDuplicated() {
				me.msgCh <- msg
			} else {
				msg.Ack(false)
			}
		}
	}
}

type Message struct {
	amqp.Delivery
	config *config.Config
	cache  gcache.Cache
}

func (me *Message) IsDuplicated() bool {
	if me.config.AutoAck {
		return false
	}
	_, err := me.cache.Get(me.DeliveryTag)
	return err == nil
}

func (me *Message) Ack(multiple bool) error {
	if me.config.AutoAck {
		return nil
	}
	if err := me.Delivery.Ack(multiple); err != nil {
		me.cache.SetWithExpire(me.DeliveryTag, true, time.Hour)
		return err
	} else {
		me.cache.Remove(me.DeliveryTag)
	}
	return nil
}
