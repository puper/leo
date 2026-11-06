package uniqid

import "errors"

// ErrSlowSubscriber 表示事件队列已满，订阅终止且不再接收后续事件。
var ErrSlowSubscriber = errors.New("uniqid: subscriber event queue is full")

// Subscription 的事件按发布顺序交付。关闭后可排空已入队事件，再检查 Err。
type Subscription struct {
	component *Component
	events    chan Event
	closed    bool
	err       error
}

// Subscribe 订阅后续事件；需要启动事件时，应在 Start 前订阅。capacity 必须为正数。
func (me *Component) Subscribe(capacity int) (*Subscription, error) {
	if capacity <= 0 {
		return nil, errors.New("uniqid: subscription capacity must be positive")
	}
	me.mu.Lock()
	defer me.mu.Unlock()
	if me.closing {
		return nil, errors.New("uniqid: component is closed")
	}
	sub := &Subscription{component: me, events: make(chan Event, capacity)}
	me.subscriptions[sub] = struct{}{}
	return sub, nil
}

func (s *Subscription) Events() <-chan Event { return s.events }

func (s *Subscription) Err() error {
	s.component.mu.RLock()
	defer s.component.mu.RUnlock()
	return s.err
}

// Close 取消此订阅；重复调用保留最初的终止错误。
func (s *Subscription) Close() error {
	s.component.mu.Lock()
	defer s.component.mu.Unlock()
	s.closeLocked(nil)
	return s.err
}

// 发送和关闭均由组件锁保护，避免并发发布时向已关闭的 channel 发送。
func (s *Subscription) closeLocked(err error) {
	if s.closed {
		return
	}
	s.closed, s.err = true, err
	delete(s.component.subscriptions, s)
	close(s.events)
}
