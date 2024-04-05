package reconnectable

import (
	"context"
	"sync"
	"time"
)

type RunFunc func() (signalCh chan struct{}, doneCh chan struct{}, err error)

func New(runFunc RunFunc, closeTimeout time.Duration, reconnectDelay time.Duration) *Component {
	ctx, cancel := context.WithCancel(context.Background())
	return &Component{
		ctx:            ctx,
		cancel:         cancel,
		wg:             &sync.WaitGroup{},
		runFunc:        runFunc,
		closeTimeout:   closeTimeout,
		reconnectDelay: reconnectDelay,
	}
}

type Component struct {
	ctx            context.Context
	cancel         context.CancelFunc
	wg             *sync.WaitGroup
	runFunc        RunFunc
	closeTimeout   time.Duration
	reconnectDelay time.Duration
}

func (me *Component) Close() error {
	me.cancel()
	me.wg.Wait()
	return me.ctx.Err()
}

func (me *Component) Start() error {
	initCh := make(chan error)
	me.wg.Add(1)
	go me.mainloop(initCh)
	return <-initCh
}

func (me *Component) mainloop(initCh chan error) {
	defer me.wg.Done()
	defer me.cancel()
	inited := false
	for {
		signalCh, doneCh, err := me.runFunc()
		if err != nil {
			if !inited {
				initCh <- err
				return
			}
		} else {
			if !inited {
				initCh <- nil
				inited = true
			}
		}
		select {
		case <-me.ctx.Done():
			close(signalCh)
			select {
			case <-doneCh:
			case <-time.After(me.closeTimeout):
			}
			return
		case <-doneCh:
			select {
			case <-me.ctx.Done():
				return
			case <-time.After(me.reconnectDelay):
			}
		}
	}
}
