package actor

import (
	"github.com/anthdm/hollywood/log"
	"reflect"
)

//

type deadLetter struct {
	logger log.Logger
	pid    *PID
}

func newDeadLetter() Receiver {
	pid := NewPID(LocalLookupAddr, "deadLetter")
	return &deadLetter{
		pid: pid,
	}
}

// Receive implements the Receiver interface, handling the deadletter messages.
// It will log the deadletter message if a logger is set. If not, it will silently
// ignore the message. Any production system should either have a logger set or provide a custom
// deadletter actor.
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
	default:
		d.logger.Errorw("unknown message arrived", "msg", msg)
	}
}
