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
		builders:  map[string]Builder{},
		instances: map[string]any{},
		config:    config,
		graph:     newGraph(),
	}
}

type Engine struct {
	mutex     sync.RWMutex
	builders  map[string]Builder
	instances map[string]any
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
	me.graph.TopologicalOrdering()
	names, err := me.graph.TopologicalOrdering()
	if err != nil {
		return err
	}
	for _, name := range names {
		if builder, ok := me.builders[name]; ok {
			var err error
			fmt.Printf("build component `%v` start\n", name)
			if me.instances[name], err = builder(); err != nil {
				return err
			}
			fmt.Printf("build component `%v` end\n", name)
		}
	}
	return nil
}

func (me *Engine) Close() error {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	return me.close()
}

func (me *Engine) close() error {
	me.graph.TopologicalOrdering()
	names, err := me.graph.TopologicalOrdering()
	if err != nil {
		return err
	}
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		if instance, ok := me.instances[name]; ok {
			fmt.Printf("close component `%v`\n", name)
			delete(me.instances, name)
			if closer, ok := instance.(Closer); ok {
				closer.Close()
			}
		}
	}
	return nil
}

func (me *Engine) GetConfig() *Config {
	return me.config
}

func (me *Engine) Get(name string) any {
	if instance, ok := me.instances[name]; ok {
		return instance
	}
	panic(fmt.Sprintf("engine: component `%v` not found", name))
}

func (me *Engine) Wait() error {
	stop := make(chan struct{})
	go func() {
		sChan := make(chan os.Signal)
		for {
			signal.Notify(sChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
			sig := <-sChan
			switch sig {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				stop <- struct{}{}
			}

		}
	}()
	<-stop
	return me.Close()
}
