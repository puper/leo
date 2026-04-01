package timewheel

import (
	"sync"
	"testing"
	"time"
)

func TestAddAndDispatch(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	jobReceived := make(chan *Job, 1)

	tw.Sub("test", func(job *Job) {
		jobReceived <- job
	})

	futureTime := time.Now().Unix() + 2
	job := &Job{
		Key:  "test",
		Id:   "id1",
		Time: futureTime,
		Data: "testdata",
	}

	tw.Add(job)

	select {
	case receivedJob := <-jobReceived:
		if receivedJob == nil {
			t.Fatal("job should be received")
		}
		if receivedJob.Data != "testdata" {
			t.Errorf("expected data 'testdata', got '%v'", receivedJob.Data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for job")
	}
}

func TestDelete(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	futureTime := time.Now().Unix() + 10
	job := &Job{
		Key:  "test",
		Id:   "id1",
		Time: futureTime,
	}

	tw.Add(job)
	tw.Delete("test", "id1")

	time.Sleep(800 * time.Millisecond)
}

func TestPurge(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	var callCount int
	tw.Sub("test", func(job *Job) {
		callCount++
	})

	for i := 0; i < 5; i++ {
		job := &Job{
			Key:  "test",
			Id:   string(rune('a' + i)),
			Time: time.Now().Unix() + 10,
		}
		tw.Add(job)
	}

	tw.Purge()

	time.Sleep(800 * time.Millisecond)

	if callCount != 0 {
		t.Errorf("Purge should not trigger callbacks, got %d calls", callCount)
	}
}

func TestConcurrentAddDelete(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			job := &Job{
				Key:  "test",
				Id:   string(rune('a' + id%26)),
				Time: time.Now().Unix() + 5,
			}
			tw.Add(job)
		}(i)
	}
	wg.Wait()

	time.Sleep(800 * time.Millisecond)
}

func TestSubscribeUnsubscribe(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	var callCount int
	var mu sync.Mutex

	tw.Sub("test", func(job *Job) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	futureTime := time.Now().Unix() + 1
	job := &Job{
		Key:  "test",
		Id:   "id1",
		Time: futureTime,
	}
	tw.Add(job)

	time.Sleep(1200 * time.Millisecond)

	tw.Unsub("test")

	mu.Lock()
	countBefore := callCount
	mu.Unlock()

	job2 := &Job{
		Key:  "test",
		Id:   "id2",
		Time: time.Now().Unix() + 1,
	}
	tw.Add(job2)

	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	countAfter := callCount
	mu.Unlock()

	if countBefore == 0 {
		t.Error("at least one callback should have been triggered")
	}
	if countAfter != countBefore {
		t.Error("after unsubscribe, no more callbacks should be triggered")
	}
}

func TestClose(t *testing.T) {
	tw := New(1000, 1000)

	var callCount int
	tw.Sub("test", func(job *Job) {
		callCount++
	})

	for i := 0; i < 10; i++ {
		job := &Job{
			Key:  "test",
			Id:   string(rune('a' + i)),
			Time: time.Now().Unix() + 1,
		}
		tw.Add(job)
	}

	time.Sleep(1500 * time.Millisecond)

	tw.Close()

	if callCount != 10 {
		t.Errorf("expected 10 callbacks, got %d", callCount)
	}
}

func TestCloseIdempotent(t *testing.T) {
	tw := New(1000, 1000)

	go func() {
		tw.Close()
	}()

	go func() {
		tw.Close()
	}()

	time.Sleep(100 * time.Millisecond)
}

func TestCloseWaitsMainloopAndDispatch(t *testing.T) {
	tw := New(1000, 1000)

	for i := 0; i < 50; i++ {
		tw.Add(&Job{
			Key:  "test",
			Id:   string(rune('a' + i%26)) + string(rune('A'+(i/26)%26)),
			Time: time.Now().Unix(),
		})
	}

	tw.Close()

	select {
	case <-tw.mainloopDone:
	default:
		t.Fatal("mainloop should have exited when Close returns")
	}

	select {
	case <-tw.done:
	default:
		t.Fatal("dispatch should have exited when Close returns")
	}
}
