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

// Scheduler is an interface that defines methods for scheduling and throughput.
type Scheduler interface {
	// Schedule schedules the given function.
	Schedule(fn func())
	// Throughput returns the throughput value.
	Throughput() int
}

// goscheduler is a type that represents a scheduler using goroutines.
type goscheduler int

// Schedule is a method of the goscheduler that schedules the given function to run as a goroutine.
func (goscheduler) Schedule(fn func()) {
	go fn()
}

// Throughput is a method of the goscheduler that returns the throughput value.
func (sched goscheduler) Throughput() int {
	return int(sched)
}

// NewScheduler is a function that creates a new scheduler with the provided throughput.
// It returns a Scheduler.
func NewScheduler(throughput int) Scheduler {
	return goscheduler(throughput)
}

// Inboxer is an interface that defines methods for sending, starting, and stopping.
type Inboxer interface {
	Send(Envelope)
	Start(Processer)
	Stop() error
}

// Inbox is a struct representing an inbox with a ring buffer, processer, scheduler, and process status.
type Inbox struct {
	rb         *ringbuffer.RingBuffer[Envelope]
	proc       Processer
	scheduler  Scheduler
	procStatus int32
}

// NewInbox is a function that creates a new inbox with the provided size.
// It returns a pointer to the Inbox.
func NewInbox(size int) *Inbox {
	return &Inbox{
		rb:        ringbuffer.New[Envelope](int64(size)),
		scheduler: NewScheduler(defaultThroughput),
	}
}

// Send is a method of the Inbox that sends the given message.
func (in *Inbox) Send(msg Envelope) {
	in.rb.Push(msg)
	in.schedule()
}

// schedule is a method of the Inbox that schedules the processing of messages.
func (in *Inbox) schedule() {
	if atomic.CompareAndSwapInt32(&in.procStatus, idle, running) {
		in.scheduler.Schedule(in.process)
	}
}

// process is a method of the Inbox that processes the messages.
func (in *Inbox) process() {
	in.run()
	atomic.StoreInt32(&in.procStatus, idle)
}

// run is a method of the Inbox that processes the messages.
// It iterates through the messages in the ring buffer, invoking the processer for each batch of messages based on the scheduler's throughput.
// If the processer status is stopped, the method stops processing.
func (in *Inbox) run() {
	i, t := 0, in.scheduler.Throughput()
	// Continue processing messages until the processer status is stopped
	for atomic.LoadInt32(&in.procStatus) != stopped {
		// If the current count exceeds the throughput, yield the processor
		if i > t {
			i = 0
			runtime.Gosched()
		}
		i++

		// Attempt to retrieve a batch of messages from the ring buffer
		if msgs, ok := in.rb.PopN(messageBatchSize); ok && len(msgs) > 0 {
			// Invoke the processer with the retrieved messages
			in.proc.Invoke(msgs)
		} else {
			// If no messages are retrieved
			return
		}
	}
}

// Start is a method of the Inbox that sets the processer for the inbox.
func (in *Inbox) Start(proc Processer) {
	in.proc = proc
}

// Stop is a method of the Inbox that stops the processing of messages by setting the processer status to stopped.
func (in *Inbox) Stop() error {
	atomic.StoreInt32(&in.procStatus, stopped)
	return nil
}
