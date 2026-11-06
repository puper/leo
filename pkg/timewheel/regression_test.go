package timewheel

import (
	"testing"
	"time"
)

func TestCompositeJobKeysDoNotCollide(t *testing.T) {
	tw := New(0, 2)
	defer tw.Close()
	received := make(chan *Job, 2)
	tw.Sub("a:b", func(j *Job) { received <- j })
	tw.Sub("a", func(j *Job) { received <- j })
	tw.Add(&Job{Key: "a:b", Id: "c", Time: time.Now().Unix()})
	tw.Add(&Job{Key: "a", Id: "b:c", Time: time.Now().Unix()})
	for i := 0; i < 2; i++ {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("distinct job lost")
		}
	}
}
func TestCallbackCanAddWithUnbufferedQueues(t *testing.T) {
	tw := New(0, 0)
	defer tw.Close()
	received := make(chan struct{}, 2)
	tw.Sub("test", func(j *Job) {
		tw.Add(&Job{Key: "later", Id: j.Id, Time: time.Now().Unix() + 100})
		received <- struct{}{}
	})
	tw.Add(&Job{Key: "test", Id: "1", Time: time.Now().Unix()})
	tw.Add(&Job{Key: "test", Id: "2", Time: time.Now().Unix()})
	for i := 0; i < 2; i++ {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("callback Add deadlocked")
		}
	}
}
