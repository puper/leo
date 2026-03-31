package mutexmanager

import (
	"sync"
	"testing"
)

func TestLockUnlockRace(t *testing.T) {
	m := New()
	key := "test-key"
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.Lock(key)
		}()
		go func() {
			defer wg.Done()
			m.Unlock(key)
		}()
	}
	wg.Wait()
}
