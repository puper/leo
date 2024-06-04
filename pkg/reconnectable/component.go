package reconnectable

import (
	"context"
	"sync"
	"time"
)

type RunFunc func(signalCh chan struct{}, doneCh chan struct{}) error

func New(runFunc RunFunc, closeTimeout time.Duration, reconnectDelay time.Duration) *Component {
	ctx, cancel := context.WithCancel(context.Background())
	return &Component{
		ctx:            ctx,
		cancel:         cancel,
		wg:             &sync.WaitGroup{},
		runFunc:        runFunc,
		closeTimeout:   closeTimeout,
		reconnectDelay: reconnectDelay,

		initCh: make(chan error, 1),
	}
}

type Component struct {
	ctx            context.Context
	cancel         context.CancelFunc
	wg             *sync.WaitGroup
	runFunc        RunFunc
	closeTimeout   time.Duration
	reconnectDelay time.Duration

	inited bool
	initCh chan error
}

func (me *Component) Close() error {
	me.cancel()
	me.wg.Wait()
	return me.ctx.Err()
}

func (me *Component) Start() error {
	me.wg.Add(1)
	go me.mainloop()
	return <-me.initCh
}

func (me *Component) mainloop() {
	defer me.wg.Done()
	defer me.cancel()
	for {
		signalCh := make(chan struct{}, 1)
		doneCh := make(chan struct{}, 1)
		go me.runFunc(signalCh, doneCh)
		// log error?
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
