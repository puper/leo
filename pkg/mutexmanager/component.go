package mutexmanager

import (
	"sync"
)

var (
	defaultMutexManager *MutexManager
	once                sync.Once
)

func Default() *MutexManager {
	once.Do(func() {
		if defaultMutexManager == nil {
			defaultMutexManager = New()
		}
	})
	return defaultMutexManager
}

type Mutex struct {
	sync.RWMutex
	locks  int64
	rlocks int64
}

func New() *MutexManager {
	return &MutexManager{
		mutexes: map[string]*Mutex{},
	}
}

type MutexManager struct {
	mutex   sync.Mutex
	mutexes map[string]*Mutex
}

func (me *MutexManager) Lock(key string) {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; !ok {
		me.mutexes[key] = &Mutex{}
	}
	me.mutexes[key].locks++
	m := me.mutexes[key]
	me.mutex.Unlock()
	m.Lock()

}

func (me *MutexManager) Unlock(key string) {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; ok {
		me.mutexes[key].Unlock()
		me.mutexes[key].locks--
		if me.mutexes[key].locks == 0 {
			delete(me.mutexes, key)
		}
		me.mutex.Unlock()
	} else {
		me.mutex.Unlock()
		panic("unlock of unlocked mutex")
	}
}

func (me *MutexManager) RLock(key string) {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; !ok {
		me.mutexes[key] = &Mutex{}
	}
	me.mutexes[key].rlocks++
	m := me.mutexes[key]
	me.mutex.Unlock()
	m.RLock()
}

func (me *MutexManager) RUnlock(key string) {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; ok {
		me.mutexes[key].RUnlock()
		me.mutexes[key].rlocks--
		if me.mutexes[key].locks == 0 && me.mutexes[key].rlocks == 0 {
			delete(me.mutexes, key)
		}
		me.mutex.Unlock()
	} else {
		me.mutex.Unlock()
		panic("r_unlock of unlocked mutex")
	}
}

func (me *MutexManager) TryLock(key string) bool {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; !ok {
		me.mutexes[key] = &Mutex{}
	}
	me.mutexes[key].locks++
	m := me.mutexes[key]
	me.mutex.Unlock()
	return m.TryLock()
}

func (me *MutexManager) TryRLock(key string) bool {
	me.mutex.Lock()
	if _, ok := me.mutexes[key]; !ok {
		me.mutexes[key] = &Mutex{}
	}
	me.mutexes[key].rlocks++
	m := me.mutexes[key]
	me.mutex.Unlock()
	return m.TryRLock()
}
