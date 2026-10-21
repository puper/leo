package subscription

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/puper/gcache"
	"github.com/puper/leo/components/rabbitmq/subscription/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultStartTimeout   = 10 * time.Second
	defaultCloseTimeout   = 10 * time.Second
	defaultReconnectDelay = time.Second
)

func New(cfg *config.Config) *Subscription {
	if cfg == nil {
		cfg = &config.Config{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscription{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		inited: false,
		initCh: make(chan error, 1),
		msgCh:  make(chan *Message, 1024),
	}
}

type Subscription struct {
	config   *config.Config
	ctx      context.Context
	cancel   context.CancelFunc
	wg       *sync.WaitGroup
	inited   bool
	initCh   chan error
	initOnce sync.Once

	subscriptionCallback func(*amqp.Channel, *config.Config, bool) (<-chan amqp.Delivery, error)

	msgCh    chan *Message
	mu       sync.Mutex
	started  bool
	closed   bool
	closeErr error
}

func (me *Subscription) Declare(ch *amqp.Channel) error {
	if me.config.ExchangeDeclare {
		if err := ch.ExchangeDeclare(
			me.config.ExchangeName,
			me.config.ExchangeType,
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("declare exchange: %w", err)
		}
	}
	if me.config.QueueDeclare {
		_, err := ch.QueueDeclare(
			me.config.QueueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("declare queue: %w", err)
		}
	}
	if me.config.QueueBind {
		err := ch.QueueBind(
			me.config.QueueName,
			me.config.RoutingKey,
			me.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("bind queue: %w", err)
		}
	}
	return nil
}

func (me *Subscription) Start() error {
	me.mu.Lock()
	if me.closed || me.started {
		me.mu.Unlock()
		return errors.New("subscription already started or closed")
	}
	me.started = true
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
	me.mu.Unlock()
	select {
	case err := <-me.initCh:
		if err != nil {
			return errors.Join(err, me.Close())
		}
		return nil
	case <-me.ctx.Done():
		return errors.Join(me.ctx.Err(), me.Close())
	case <-time.After(me.startTimeout()):
		return errors.Join(fmt.Errorf("start timeout after %v", me.startTimeout()), me.Close())
	}
}

func (me *Subscription) MsgCh() <-chan *Message {
	return me.msgCh
}

func (me *Subscription) Close() error {
	me.mu.Lock()
	me.closed = true
	me.cancel()
	me.mu.Unlock()
	me.wg.Wait()
	me.mu.Lock()
	defer me.mu.Unlock()
	return me.closeErr
}

// 只能在 onConnected 中调用。
func (me *Subscription) ClearMsgs() {
	for {
		select {
		case <-me.msgCh:
		default:
			return
		}
	}
}
func (me *Subscription) mainloop() {
	defer me.wg.Done()
	defer me.cancel()
	reconnectDelay := me.reconnectDelay()
	for {
		signalCh := make(chan struct{}, 1)
		doneCh := make(chan struct{}, 1)
		go me.run(signalCh, doneCh)
		select {
		case <-me.ctx.Done():
			close(signalCh)
			select {
			case <-doneCh:
			case <-time.After(me.closeTimeout()):
				me.mu.Lock()
				me.closeErr = context.DeadlineExceeded
				me.mu.Unlock()
			}
			return
		case <-doneCh:
			select {
			case <-me.ctx.Done():
				return
			case <-time.After(reconnectDelay):
			}
			reconnectDelay = me.nextReconnectDelay(reconnectDelay)
		}
	}
}

func (me *Subscription) nextReconnectDelay(currentDelay time.Duration) time.Duration {
	nextDelay := currentDelay * 2
	if nextDelay > time.Minute {
		return time.Minute
	}
	return nextDelay
}

func (me *Subscription) run(signalCh chan struct{}, doneCh chan struct{}) error {
	defer close(doneCh)
	uri, err := amqp.ParseURI(me.config.Addr)
	if err != nil {
		me.notifyInit(err)
		return err
	}
	timeout := 30 * time.Second
	if uri.ConnectionTimeout > 0 {
		timeout = time.Duration(uri.ConnectionTimeout) * time.Millisecond
	}
	var raw net.Conn
	var stopCancel func() bool
	defer func() {
		if stopCancel != nil {
			stopCancel()
		}
		if raw != nil {
			raw.Close()
		}
	}()
	conn, err := amqp.DialConfig(me.config.Addr, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			c, err := (&net.Dialer{Timeout: timeout}).DialContext(me.ctx, network, addr)
			if err != nil {
				return nil, err
			}
			raw = c
			stopCancel = context.AfterFunc(me.ctx, func() { c.Close() })
			if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	})
	if err != nil {
		me.notifyInit(err)
		return err
	}
	defer func() { raw.Close(); conn.Close() }()
	connCloseCh := make(chan *amqp.Error, 1)
	conn.NotifyClose(connCloseCh)
	ch, err := conn.Channel()
	if err != nil {
		me.notifyInit(err)
		return err
	}
	// DeliveryTag 仅在当前 channel 中唯一，旧消息持有旧缓存，不能污染重连后的缓存。
	cache := gcache.New(10000).LRU().Build()
	if me.config.PrefetchCount > 0 || me.config.PrefetchSize > 0 {
		if err := ch.Qos(me.config.PrefetchCount, me.config.PrefetchSize, false); err != nil {
			me.notifyInit(err)
			return fmt.Errorf("set qos: %w", err)
		}
	}
	chClosedCh := make(chan *amqp.Error, 1)
	ch.NotifyClose(chClosedCh)
	chCancelCh := make(chan string, 1)
	ch.NotifyCancel(chCancelCh)
	if !me.inited {
		if err := me.Declare(ch); err != nil {
			me.notifyInit(err)
			return err
		}
	}
	deliveries, err := me.subscriptionCallback(ch, me.config, me.inited)
	if err != nil {
		me.notifyInit(err)
		return err
	}
	if !me.inited {
		me.inited = true
		me.notifyInit(nil)
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
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("deliveries channel closed")
			}
			if err := me.forwardDelivery(d, cache, signalCh); err != nil {
				return err
			}
		}
	}
}

func (me *Subscription) forwardDelivery(d amqp.Delivery, cache gcache.Cache, signalCh <-chan struct{}) error {
	msg := &Message{Delivery: d, config: me.config, cache: cache}
	if msg.IsDuplicated() {
		return msg.Ack(false)
	}
	// 缓冲区满时保留背压；未处理消息不能通过 Ack 丢弃。
	select {
	case me.msgCh <- msg:
		return nil
	case <-signalCh:
		return context.Canceled
	case <-me.ctx.Done():
		return me.ctx.Err()
	}
}

func (me *Subscription) notifyInit(err error) {
	me.initOnce.Do(func() {
		select {
		case me.initCh <- err:
		default:
		}
	})
}

func (me *Subscription) startTimeout() time.Duration {
	if me.config.StartTimeout > 0 {
		return me.config.StartTimeout
	}
	return defaultStartTimeout
}

func (me *Subscription) closeTimeout() time.Duration {
	if me.config.CloseTimeout > 0 {
		return me.config.CloseTimeout
	}
	return defaultCloseTimeout
}

func (me *Subscription) reconnectDelay() time.Duration {
	if me.config.ReconnectDelay > 0 {
		return me.config.ReconnectDelay
	}
	return defaultReconnectDelay
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
		return err
	}
	me.cache.SetWithExpire(me.DeliveryTag, true, time.Hour)
	return nil
}
