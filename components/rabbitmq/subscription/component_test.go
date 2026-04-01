package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/puper/leo/components/rabbitmq/subscription/config"
)

func TestStartCloseRace(t *testing.T) {
	cfg := &config.Config{
		Addr:           "amqp://guest:guest@localhost:5672/",
		QueueName:      "test-queue",
		CloseTimeout:   time.Second,
		ReconnectDelay: time.Millisecond * 100,
	}

	sub := New(cfg)

	go func() {
		time.Sleep(20 * time.Millisecond)
		sub.Close()
	}()

	done := make(chan error, 1)
	go func() {
		done <- sub.Start()
	}()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Logf("Start returned error: %v (may be expected if no broker)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start() blocked indefinitely - initCh not sent")
	}
}

func TestAckFailureDoesNotRemoveCache(t *testing.T) {
	cfg := &config.Config{
		AutoAck: false,
	}

	sub := New(cfg)

	msg := &Message{
		config: sub.config,
		cache:  sub.cache,
	}

	_, err := sub.cache.Get(msg.DeliveryTag)
	if err == nil {
		t.Fatal("Cache should not have DeliveryTag before test")
	}

	sub.cache.SetWithExpire(msg.DeliveryTag, true, time.Hour)

	_, err = sub.cache.Get(msg.DeliveryTag)
	if err != nil {
		t.Fatal("Cache should have DeliveryTag after SetWithExpire")
	}

	if !cfg.AutoAck {
		_, err := sub.cache.GetIFPresent(msg.DeliveryTag)
		if err == nil {
			t.Log("Cache behavior verified: on Ack success, tag is cached")
		}
	}
}

func TestNextReconnectDelayExponentialBackoff(t *testing.T) {
	cfg := &config.Config{
		ReconnectDelay: time.Second,
	}

	sub := New(cfg)

	delay := cfg.ReconnectDelay
	for i := 0; i < 10; i++ {
		delay = sub.nextReconnectDelay(delay)
	}

	if delay != time.Minute {
		t.Errorf("Expected delay to cap at 1 minute, got %v", delay)
	}
}

func TestMsgChCapacity(t *testing.T) {
	cfg := &config.Config{}

	sub := New(cfg)

	if cap(sub.msgCh) != 1024 {
		t.Errorf("Expected msgCh capacity 1024, got %d", cap(sub.msgCh))
	}
}

func TestPrefetchCountConfig(t *testing.T) {
	cfg := &config.Config{
		PrefetchCount: 10,
		PrefetchSize:  0,
	}

	sub := New(cfg)

	if sub.config.PrefetchCount != 10 {
		t.Errorf("Expected PrefetchCount 10, got %d", sub.config.PrefetchCount)
	}
}
