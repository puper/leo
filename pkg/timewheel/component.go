package timewheel

import (
	"sync"
	"time"
)

var (
	tw          *TimeWheel
	once        sync.Once
	requestPool = sync.Pool{
		New: func() any {
			return &request{}
		},
	}
)

func Default() *TimeWheel {
	once.Do(func() {
		if tw == nil {
			tw = New(10000, 10000)
		}
	})
	return tw
}

type Job struct {
	Key  string
	Id   string
	Time int64
	Data any
}

type request struct {
	action int
	job    *Job
}

func New(reqLen, dispatchLen int) *TimeWheel {
	me := &TimeWheel{
		jobsByTime:     map[int64]map[string]*Job{},
		jobsById:       map[string]*Job{},
		reqs:           make(chan *request, reqLen),
		dispatchJobs:   make(chan *Job, dispatchLen),
		closed:         make(chan struct{}),
		dispatchClosed: make(chan struct{}),
		done:           make(chan struct{}),
	}
	go me.mainloop()
	go me.dispatch()
	return me
}

type Callback func(*Job)

type TimeWheel struct {
	jobsByTime     map[int64]map[string]*Job
	jobsById       map[string]*Job
	reqs           chan *request
	dispatchJobs   chan *Job
	closed         chan struct{}
	dispatchClosed chan struct{}
	done           chan struct{}
	closeOnce      sync.Once
	callbacks      sync.Map
}

func (me *TimeWheel) Close() {
	me.closeOnce.Do(func() {
		close(me.closed)
		close(me.dispatchClosed)
	})
	<-me.done
}

func (me *TimeWheel) Sub(key string, f Callback) {
	me.callbacks.Store(key, f)
}

func (me *TimeWheel) Unsub(key string) {
	me.callbacks.Delete(key)
}

func (me *TimeWheel) dispatch() {
	for {
		select {
		case job := <-me.dispatchJobs:
			if f, ok := me.callbacks.Load(job.Key); ok {
				f.(Callback)(job)
			}
		case <-me.dispatchClosed:
			for {
				select {
				case job := <-me.dispatchJobs:
					if f, ok := me.callbacks.Load(job.Key); ok {
						f.(Callback)(job)
					}
				default:
					close(me.done)
					return
				}
			}
		}
	}
}

func (me *TimeWheel) Add(job *Job) {
	select {
	case <-me.closed:
		return
	case me.reqs <- getRequest(0, job):
	}
}

func (me *TimeWheel) Delete(key, id string) {
	select {
	case <-me.closed:
		return
	case me.reqs <- getRequest(1, &Job{Id: id, Key: key}):
	}
}

func (me *TimeWheel) Purge() {
	select {
	case <-me.closed:
		return
	case me.reqs <- getRequest(2, nil):
	}
}

func getRequest(action int, job *Job) *request {
	req := requestPool.Get().(*request)
	req.action = action
	req.job = job
	return req
}

func (me *TimeWheel) mainloop() {
	tk := time.NewTicker(time.Millisecond * 600)
	lastTime := time.Now().Unix()
	expiredJobTimes := map[int64]struct{}{}
LOOP:
	for {
		select {
		case now := <-tk.C:
			for jobTime := range expiredJobTimes {
				for _, job := range me.jobsByTime[jobTime] {
					mapKey := job.Key + ":" + job.Id
					delete(me.jobsById, mapKey)
					me.dispatchJobs <- job
				}
				delete(me.jobsByTime, jobTime)
			}
			expiredJobTimes = map[int64]struct{}{}
			for jobTime := lastTime + 1; jobTime <= now.Unix(); jobTime++ {
				for _, job := range me.jobsByTime[jobTime] {
					mapKey := job.Key + ":" + job.Id
					delete(me.jobsById, mapKey)
					me.dispatchJobs <- job
				}
				delete(me.jobsByTime, jobTime)
			}
			if now.Unix() > lastTime {
				lastTime = now.Unix()
			}
		case req := <-me.reqs:
			mapKey := req.job.Key + ":" + req.job.Id
			if req.action == 0 {
				if job, ok := me.jobsById[mapKey]; ok {
					delete(me.jobsByTime[job.Time], mapKey)
				}
				me.jobsById[mapKey] = req.job
				if _, ok := me.jobsByTime[req.job.Time]; !ok {
					me.jobsByTime[req.job.Time] = map[string]*Job{}
				}
				me.jobsByTime[req.job.Time][mapKey] = req.job
				if req.job.Time <= lastTime {
					expiredJobTimes[req.job.Time] = struct{}{}
				}
			} else if req.action == 1 {
				if job, ok := me.jobsById[mapKey]; ok {
					delete(me.jobsById, mapKey)
					delete(me.jobsByTime[job.Time], mapKey)
				}
			} else if req.action == 2 {
				me.jobsByTime = map[int64]map[string]*Job{}
				me.jobsById = map[string]*Job{}
			}
			req.action = 0
			req.job = nil
			requestPool.Put(req)
		case <-me.closed:
			break LOOP

		}
	}
	tk.Stop()
}
