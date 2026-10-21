package subscription

import (
	"errors"
	"github.com/puper/gcache"
	"github.com/puper/leo/components/rabbitmq/subscription/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"testing"
	"time"
)

type ackResult struct {
	err  error
	acks chan uint64
}

func (a *ackResult) Ack(tag uint64, _ bool) error {
	if a.err != nil {
		return a.err
	}
	a.acks <- tag
	return nil
}
func (a *ackResult) Nack(uint64, bool, bool) error { return nil }
func (a *ackResult) Reject(uint64, bool) error     { return nil }
func TestAckFailureIsNotCached(t *testing.T) {
	cache := gcache.New(10).LRU().Build()
	want := errors.New("ack failed")
	ack := &ackResult{err: want, acks: make(chan uint64, 1)}
	msg := &Message{Delivery: amqp.Delivery{DeliveryTag: 1, Acknowledger: ack}, config: &config.Config{}, cache: cache}
	if err := msg.Ack(false); !errors.Is(err, want) {
		t.Fatal(err)
	}
	if msg.IsDuplicated() {
		t.Fatal("failed ack cached")
	}
	ack.err = nil
	if err := msg.Ack(false); err != nil {
		t.Fatal(err)
	}
	if !msg.IsDuplicated() {
		t.Fatal("successful ack not cached")
	}
}
func TestFullQueueBackpressuresWithoutAck(t *testing.T) {
	sub := New(&config.Config{})
	defer sub.Close()
	sub.msgCh = make(chan *Message, 1)
	sub.msgCh <- &Message{}
	ack := &ackResult{acks: make(chan uint64, 1)}
	done := make(chan error, 1)
	go func() {
		done <- sub.forwardDelivery(amqp.Delivery{DeliveryTag: 1, Acknowledger: ack}, gcache.New(10).LRU().Build(), nil)
	}()
	select {
	case err := <-done:
		t.Fatalf("full queue did not apply backpressure: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	select {
	case <-ack.acks:
		t.Fatal("unprocessed delivery acknowledged")
	default:
	}
	<-sub.msgCh
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("delivery did not resume")
	}
	if msg := <-sub.msgCh; msg.DeliveryTag != 1 {
		t.Fatal("wrong delivery")
	}
}
func TestDeliveryTagsAreIsolatedBetweenChannels(t *testing.T) {
	sub := New(&config.Config{})
	defer sub.Close()
	ack := &ackResult{acks: make(chan uint64, 2)}
	oldCache, newCache := gcache.New(10).LRU().Build(), gcache.New(10).LRU().Build()
	old := &Message{Delivery: amqp.Delivery{DeliveryTag: 1, Acknowledger: ack}, config: sub.config, cache: oldCache}
	if err := old.Ack(false); err != nil {
		t.Fatal(err)
	}
	if err := sub.forwardDelivery(amqp.Delivery{DeliveryTag: 1, Acknowledger: ack}, newCache, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-sub.msgCh:
		if msg.IsDuplicated() {
			t.Fatal("new channel inherited old acknowledgement")
		}
	case <-time.After(time.Second):
		t.Fatal("new channel delivery dropped")
	}
}
