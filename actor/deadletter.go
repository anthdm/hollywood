package actor

import (
	reflect "reflect"

	"github.com/anthdm/hollywood/log"
)

type deadLetter struct {
	eventStream *EventStream
	pid         *PID
}

func newDeadLetter(eventStream *EventStream) *deadLetter {
	return &deadLetter{
		eventStream: eventStream,
		pid:         NewPID(localLookupAddr, "deadLetter"),
	}
}

func (d *deadLetter) Send(dest *PID, msg any, sender *PID) {
	log.Warnw("[DEADLETTER]", log.M{
		"dest":   dest,
		"msg":    reflect.TypeOf(msg),
		"sender": sender,
	})
	d.eventStream.Publish(&DeadLetterEvent{
		Target:  dest,
		Message: msg,
		Sender:  sender,
	})
}

func (d *deadLetter) PID() *PID { return d.pid }
func (d *deadLetter) Shutdown() {}
