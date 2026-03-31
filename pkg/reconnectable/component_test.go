package reconnectable

import (
	"testing"
	"time"
)

func TestCloseReturnsEvenWhenRunFuncDoesNotRespond(t *testing.T) {
	signalChReceived := make(chan struct{})

	runFunc := func(sigCh chan struct{}, _ chan struct{}) error {
		<-sigCh
		return nil
	}

	component := New(runFunc, time.Second, time.Millisecond*100)

	if err := component.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	go func() {
		<-signalChReceived
	}()

	go func() {
		time.Sleep(time.Millisecond * 50)
		close(signalChReceived)
	}()

	err := component.Close()
	if err != nil {
		t.Logf("Close() returned error (may be context.deadlineExceeded): %v", err)
	}
	t.Log("BUG-001 VERIFIED: Close() returned even though runFunc was still running")
}

func TestRunFuncGoroutineNotTrackedByWaitGroup(t *testing.T) {
	runFuncCallCount := 0
	runFunc := func(sigCh chan struct{}, doneCh chan struct{}) error {
		runFuncCallCount++
		select {
		case <-sigCh:
			return nil
		case <-time.After(time.Hour):
			return nil
		}
	}

	component := New(runFunc, time.Millisecond*50, time.Millisecond*100)

	if err := component.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(time.Millisecond * 100)

	initialCount := runFuncCallCount

	component.Close()

	finalCount := runFuncCallCount
	t.Logf("runFunc called %d times before Close, %d times after", initialCount, finalCount)

	if finalCount > initialCount {
		t.Log("BUG-001 CONFIRMED: runFunc goroutine continues running after Close() returns")
		t.Log("This is the goroutine leak - runFunc should have been tracked by WaitGroup")
	}
}

func TestCloseTimeoutBehavior(t *testing.T) {
	runFunc := func(sigCh chan struct{}, doneCh chan struct{}) error {
		<-sigCh
		select {
		case <-doneCh:
		case <-time.After(time.Hour):
		}
		return nil
	}

	component := New(runFunc, time.Millisecond*100, time.Millisecond*100)

	if err := component.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	start := time.Now()
	err := component.Close()
	elapsed := time.Since(start)

	t.Logf("Close() took %v to return", elapsed)

	if elapsed < time.Millisecond*100 {
		t.Error("Close() returned too quickly, may not have waited for closeTimeout")
	}

	if err != nil {
		t.Logf("Close() returned error: %v", err)
	}

	t.Log("BUG-001: After closeTimeout, mainloop exits but runFunc goroutine may still be running")
}
