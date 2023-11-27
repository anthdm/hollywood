package actor

import (
	"github.com/anthdm/hollywood/log"
	"reflect"
)

//

type deadLetter struct {
	logger log.Logger
	pid    *PID
	msgs   []*DeadLetterEvent
}

func newDeadLetter() Receiver {
	pid := NewPID(LocalLookupAddr, "deadLetter")
	msgs := make([]*DeadLetterEvent, 0)
	return &deadLetter{
		msgs: msgs,
		pid:  pid,
	}
}

// Receive implements the Receiver interface, handling the deadletter messages.
func (d *deadLetter) Receive(ctx *Context) {
	switch msg := ctx.Message().(type) {
	case Started:
		// intialize logger on deadletter startup. is this a sane approach? I'm not sure how the get to the logger otherwise.
		d.logger = ctx.Engine().logger.SubLogger("[deadletter]")
		d.logger.Debugw("default deadletter actor started")
	case Stopped:
		d.logger.Debugw("default deadletter actor stopped")
	case Initialized:
		d.logger.Debugw("default deadletter actor initialized")
	case *DeadLetterEvent:
		d.logger.Warnw("deadletter arrived", "msg-type", reflect.TypeOf(msg),
			"sender", msg.Sender, "target", msg.Target, "msg", msg.Message)
		d.msgs = append(d.msgs, msg)
	default:
		d.logger.Errorw("unknown message arrived", "msg", msg)
	}
}
