package engine

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/viper"
)

type Config = viper.Viper

type Builder func() (any, error)

type Closer interface {
	Close() error
}

func New(config *Config) *Engine {
	return &Engine{
		builders: map[string]Builder{},
		config:   config,
		graph:    newGraph(),
	}
}

type Engine struct {
	mutex     sync.RWMutex
	builders  map[string]Builder
	instances sync.Map
	config    *Config
	graph     *graph
}

func (me *Engine) Register(name string, builder Builder, dependencies ...string) {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	if _, ok := me.builders[name]; !ok {
		me.builders[name] = builder
		me.graph.AddVertex(name)
		for _, dependency := range dependencies {
			me.graph.AddEdge(dependency, name)
		}
	}
}

func (me *Engine) Build() error {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	names, err := me.graph.TopologicalOrdering()
	if err != nil {
		return err
	}
	for _, name := range names {
		builder, ok := me.builders[name]
		if !ok {
			return fmt.Errorf("engine: builder `%s` not registered", name)
		}
		if builder == nil {
			return fmt.Errorf("engine: builder `%s` is nil", name)
		}
		instance, err := builder()
		if err != nil {
			return err
		}
		me.instances.Store(name, instance)
	}
	return nil
}

func (me *Engine) Close() error {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	return me.close()
}

func (me *Engine) close() error {
	names, err := me.graph.TopologicalOrdering()
	if err != nil {
		return err
	}
	var closeErrors []error
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		instance, ok := me.instances.Load(name)
		if ok {
			me.instances.Delete(name)
			if closer, ok := instance.(Closer); ok {
				if err := closer.Close(); err != nil {
					closeErrors = append(closeErrors, err)
				}
			}
		}
	}
	if len(closeErrors) > 0 {
		return closeErrors[0]
	}
	return nil
}

func (me *Engine) GetConfig() *Config {
	return me.config
}

func (me *Engine) Get(name string) any {
	if instance, ok := me.instances.Load(name); ok {
		return instance
	}
	panic(fmt.Sprintf("engine: component `%v` not found", name))
}

func (me *Engine) Wait() error {
	stop := make(chan struct{})
	go func() {
		sChan := make(chan os.Signal, 1)
		signal.Notify(sChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			sig := <-sChan
			switch sig {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				stop <- struct{}{}
				return
			}
		}
	}()
	<-stop
	return me.Close()
}
