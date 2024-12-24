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
	stopped int32 = iota
	starting
	idle
	running
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
		rb:         ringbuffer.New[Envelope](int64(size)),
		scheduler:  NewScheduler(defaultThroughput),
		procStatus: stopped,
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
	if atomic.CompareAndSwapInt32(&in.procStatus, running, idle) && in.rb.Len() > 0 {
		// messages might have been added to the ring-buffer between the last pop and the transition to idle.
		// if this is the case, then we should schedule again
		in.schedule()
	}
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
	// transition to "starting" and then "idle" to ensure no race condition on in.proc
	if atomic.CompareAndSwapInt32(&in.procStatus, stopped, starting) {
		in.proc = proc
		atomic.SwapInt32(&in.procStatus, idle)
		in.schedule()
	}
}

func (in *Inbox) Stop() error {
	atomic.StoreInt32(&in.procStatus, stopped)
	return nil
}
