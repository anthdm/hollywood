package actor

import (
	"runtime"
	"sync/atomic"

	"github.com/anthdm/hollywood/ringbuffer"
)

const (
	defaultThroughput = 300
	messageBatchSize  = 1024 * 4
)

const (
	idle int32 = iota
	running
	stopped
)

type Scheduler interface {
	Schedule(fn func())
	Throughput() int
}

type goscheduler int

func (goscheduler) Schedule(fn func()) {
	go fn()
}

func (sched goscheduler) Throughput() int {
	return int(sched)
}

func NewScheduler(throughput int) Scheduler {
	return goscheduler(throughput)
}

type Inboxer interface {
	Send(Envelope)
	Start(Processer)
	Stop() error
}

type Inbox struct {
	rb         *ringbuffer.RingBuffer[Envelope]
	proc       Processer
	scheduler  Scheduler
	procStatus int32
}

func NewInbox(size int) *Inbox {
	return &Inbox{
		rb:        ringbuffer.New[Envelope](int64(size)),
		scheduler: NewScheduler(defaultThroughput),
	}
}

func (in *Inbox) Send(msg Envelope) {
	in.rb.Push(msg)
	in.schedule()
}

func (in *Inbox) schedule() {
	if atomic.CompareAndSwapInt32(&in.procStatus, idle, running) {
		in.scheduler.Schedule(in.process)
	}
}

func (in *Inbox) process() {
	in.run()
	atomic.StoreInt32(&in.procStatus, idle)
}

func (in *Inbox) run() {
	i, t := 0, in.scheduler.Throughput()
	for atomic.LoadInt32(&in.procStatus) != stopped {
		if i > t {
			i = 0
			runtime.Gosched()
		}
		i++

		if msgs, ok := in.rb.PopN(messageBatchSize); ok && len(msgs) > 0 {
			in.proc.Invoke(msgs)
		} else {
			return
		}
	}
}

func (in *Inbox) Start(proc Processer) {
	in.proc = proc
}

func (in *Inbox) Stop() error {
	atomic.StoreInt32(&in.procStatus, stopped)
	return nil
}
