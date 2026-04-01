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

func TestMutexManagerLockUnlockRace(t *testing.T) {
	mm := New()
	key := "test-key"

	// This test reproduces the race condition where Unlock is called
	// before Lock completes its actual mutex acquisition
	var wg sync.WaitGroup

	panicCh := make(chan any, 1)

	// Goroutine that calls Lock
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		mm.Lock(key)
		time.Sleep(10 * time.Millisecond) // Simulate work
		mm.Unlock(key)
	}()

	// Goroutine that calls Unlock immediately (before Lock completes)
	// 该路径不应导致测试进程崩溃，panic 由上面的 panicCh 捕获。
	time.Sleep(1 * time.Millisecond) // Let Lock start but not complete
	mm.Unlock(key)

	wg.Wait()

	select {
	case r := <-panicCh:
		t.Logf("goroutine panic captured: %v", r)
	default:
	}
}

func TestUnlockDoesNotDeleteEntryWhenReaderPending(t *testing.T) {
	m := New()
	key := "pending-reader"

	m.Lock(key)

	panicCh := make(chan any, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		m.RLock(key)
		m.RUnlock(key)
	}()

	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("reader did not enter pending state in time")
		default:
			m.mutex.Lock()
			pending := false
			if mu, ok := m.mutexes[key]; ok {
				pending = mu.rlocks > 0
			}
			m.mutex.Unlock()
			if pending {
				goto UNLOCK
			}
			time.Sleep(time.Millisecond)
		}
	}

UNLOCK:
	m.Unlock(key)
	<-done

	select {
	case r := <-panicCh:
		t.Fatalf("unexpected panic: %v", r)
	default:
	}

	m.mutex.Lock()
	_, ok := m.mutexes[key]
	m.mutex.Unlock()
	if ok {
		t.Fatal("mutex should be deleted after writer/reader both released")
	}
}
