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
	received := make(chan *Job, 2)
	tw.Sub("test", func(job *Job) { received <- job })
	futureTime := time.Now().Unix() + 2
	job := &Job{
		Key:  "test",
		Id:   "id1",
		Time: futureTime,
	}

	tw.Add(job)
	tw.Delete("test", "id1")
	tw.Add(&Job{Key: "test", Id: "sentinel", Time: futureTime})
	assertOnlySentinel(t, tw, received)
}

func TestPurge(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()

	received := make(chan *Job, 6)
	tw.Sub("test", func(job *Job) { received <- job })
	futureTime := time.Now().Unix() + 2
	for i := 0; i < 5; i++ {
		job := &Job{
			Key:  "test",
			Id:   string(rune('a' + i)),
			Time: futureTime,
		}
		tw.Add(job)
	}

	tw.Purge()
	tw.Add(&Job{Key: "test", Id: "sentinel", Time: futureTime})
	assertOnlySentinel(t, tw, received)
}

func TestConcurrentAddDelete(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()
	received := make(chan *Job, 101)
	tw.Sub("test", func(job *Job) { received <- job })
	futureTime := time.Now().Unix() + 2
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			job := &Job{
				Key:  "test",
				Id:   string(rune('a' + id)),
				Time: futureTime,
			}
			tw.Add(job)
			tw.Delete(job.Key, job.Id)
		}(i)
	}
	wg.Wait()
	tw.Add(&Job{Key: "test", Id: "sentinel", Time: futureTime})
	assertOnlySentinel(t, tw, received)
}

// 哨兵与被删除任务同一到期时间，防止在任务尚未到期时就误判删除成功。
func assertOnlySentinel(t *testing.T, tw *TimeWheel, received chan *Job) {
	t.Helper()
	select {
	case job := <-received:
		if job.Id != "sentinel" {
			t.Fatalf("removed job was dispatched: %s", job.Id)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sentinel was not dispatched")
	}
	tw.Close()
	select {
	case job := <-received:
		t.Fatalf("unexpected job after sentinel: %s", job.Id)
	default:
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()
	received := make(chan *Job, 2)
	tw.Sub("test", func(job *Job) { received <- job })
	futureTime := time.Now().Unix() + 1
	job := &Job{
		Key:  "test",
		Id:   "id1",
		Time: futureTime,
	}
	tw.Add(job)

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("subscribed callback was not dispatched")
	}
	tw.Unsub("test")
	futureTime = time.Now().Unix() + 2
	job2 := &Job{
		Key:  "test",
		Id:   "id2",
		Time: futureTime,
	}
	tw.Add(job2)
	tw.Sub("sentinel", func(job *Job) { received <- job })
	tw.Add(&Job{Key: "sentinel", Id: "sentinel", Time: futureTime})
	assertOnlySentinel(t, tw, received)
}

func TestClose(t *testing.T) {
	tw := New(1000, 1000)
	defer tw.Close()
	received := make(chan struct{}, 10)
	tw.Sub("test", func(job *Job) { received <- struct{}{} })

	for i := 0; i < 10; i++ {
		job := &Job{
			Key:  "test",
			Id:   string(rune('a' + i)),
			Time: time.Now().Unix() + 1,
		}
		tw.Add(job)
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for i := 0; i < 10; i++ {
		select {
		case <-received:
		case <-timer.C:
			t.Fatalf("received %d callbacks, want 10", i)
		}
	}
	tw.Close()
}

func TestCloseIdempotent(t *testing.T) {
	tw := New(1000, 1000)
	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			tw.Close()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent Close did not return")
		}
	}
}

func TestCloseWaitsMainloopAndDispatch(t *testing.T) {
	tw := New(1000, 1000)

	for i := 0; i < 50; i++ {
		tw.Add(&Job{
			Key:  "test",
			Id:   string(rune('a'+i%26)) + string(rune('A'+(i/26)%26)),
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
