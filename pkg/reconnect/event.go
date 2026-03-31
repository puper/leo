package reconnect

import "time"

type EventHandler interface {
	OnConnected()
	OnDisconnected(err error)
	OnReconnecting(attempt int, delay time.Duration)
	OnError(err error)
}

type NopEventHandler struct{}

func (h *NopEventHandler) OnConnected()                                    {}
func (h *NopEventHandler) OnDisconnected(err error)                        {}
func (h *NopEventHandler) OnReconnecting(attempt int, delay time.Duration) {}
func (h *NopEventHandler) OnError(err error)                               {}

type EventHandlers []EventHandler

func (ehs EventHandlers) OnConnected() {
	for _, h := range ehs {
		h.OnConnected()
	}
}

func (ehs EventHandlers) OnDisconnected(err error) {
	for _, h := range ehs {
		h.OnDisconnected(err)
	}
}

func (ehs EventHandlers) OnReconnecting(attempt int, delay time.Duration) {
	for _, h := range ehs {
		h.OnReconnecting(attempt, delay)
	}
}

func (ehs EventHandlers) OnError(err error) {
	for _, h := range ehs {
		h.OnError(err)
	}
}
