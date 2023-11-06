package actor

import (
	"reflect"
	"sync"

	"github.com/anthdm/hollywood/log"
)

// TODO: The deadLetter is implemented as a plain Processer, but
// can actually be implemented as a Receiver. This is a good first issue.

type deadLetter struct {
	eventStream *EventStream
	pid         *PID
	logger      log.Logger
}

func newDeadLetter(eventStream *EventStream) *deadLetter {
	return &deadLetter{
		eventStream: eventStream,
		pid:         NewPID(LocalLookupAddr, "deadLetter"),
	}
}

func (d *deadLetter) Send(dest *PID, msg any, sender *PID) {
	d.logger.Warnw("Send",
		"dest", dest,
		"msg", reflect.TypeOf(msg),
		"sender", sender,
	)
	d.eventStream.Publish(&DeadLetterEvent{
		Target:  dest,
		Message: msg,
		Sender:  sender,
	})
}

func (d *deadLetter) PID() *PID                  { return d.pid }
func (d *deadLetter) Shutdown(_ *sync.WaitGroup) {}
func (d *deadLetter) Start()                     {}
func (d *deadLetter) Invoke([]Envelope)          {}
