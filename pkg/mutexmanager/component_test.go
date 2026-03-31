package mutexmanager

import (
	"sync"
	"testing"
	"time"
)

func TestLockUnlock(t *testing.T) {
	m := New()
	key := "test-lock"

	m.Lock(key)
	m.Unlock(key)

	if _, ok := m.mutexes[key]; ok {
		t.Fatal("mutex should be deleted after unlock")
	}
}

func TestLockUnlockDifferentKeys(t *testing.T) {
	m := New()

	m.Lock("key1")
	m.Lock("key2")
	m.Unlock("key1")
	m.Unlock("key2")

	if len(m.mutexes) != 0 {
		t.Fatal("all mutexes should be deleted")
	}
}

func TestRLock(t *testing.T) {
	m := New()
	key := "test-rlock"

	m.RLock(key)
	m.RUnlock(key)

	if _, ok := m.mutexes[key]; ok {
		t.Fatal("mutex should be deleted after all unlocks")
	}
}

func TestRLockConcurrent(t *testing.T) {
	m := New()
	key := "test-rlock-concurrent"
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RLock(key)
			time.Sleep(10 * time.Millisecond)
			m.RUnlock(key)
		}()
	}
	wg.Wait()

	if _, ok := m.mutexes[key]; ok {
		t.Fatal("mutex should be deleted after all unlocks")
	}
}

func TestLockAndRLock互斥(t *testing.T) {
	m := New()
	key := "test-mutex"

	m.RLock(key)

	done := make(chan bool, 1)
	go func() {
		m.Lock(key)
		done <- true
	}()

	select {
	case <-done:
		t.Fatal("Lock should block when RLock is held")
	case <-time.After(50 * time.Millisecond):
	}

	m.RUnlock(key)
	<-done
}

func TestTryLock(t *testing.T) {
	m := New()
	key := "test-trylock"

	ok := m.TryLock(key)
	if !ok {
		t.Fatal("TryLock should succeed on unlocked mutex")
	}

	ok = m.TryLock(key)
	if ok {
		t.Fatal("TryLock should fail on locked mutex")
	}

	m.Unlock(key)
}

func TestTryRLock(t *testing.T) {
	m := New()
	key := "test-tryrlock"

	ok := m.TryRLock(key)
	if !ok {
		t.Fatal("TryRLock should succeed on unlocked mutex")
	}

	ok = m.TryRLock(key)
	if !ok {
		t.Fatal("TryRLock should succeed when already holding read lock")
	}

	m.RUnlock(key)
	m.RUnlock(key)
}

func TestLockTimeout成功(t *testing.T) {
	m := New()
	key := "test-lock-timeout"

	ok := m.LockTimeout(key, time.Second)
	if !ok {
		t.Fatal("LockTimeout should succeed on unlocked mutex")
	}
	m.Unlock(key)
}

func TestRLockTimeout成功(t *testing.T) {
	m := New()
	key := "test-rlock-timeout"

	ok := m.RLockTimeout(key, time.Second)
	if !ok {
		t.Fatal("RLockTimeout should succeed on unlocked mutex")
	}
	m.RUnlock(key)
}
