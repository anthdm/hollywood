package actor

import (
	"log/slog"
	"reflect"
)

//

type deadLetter struct {
	pid *PID
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
		slog.Debug("default deadletter actor started")
	case Stopped:
		slog.Debug("default deadletter actor stopped")
	case Initialized:
		slog.Debug("default deadletter actor initialized")
	case *DeadLetterEvent:
		slog.Warn("deadletter arrived", "msg-type", reflect.TypeOf(msg),
			"sender", msg.Sender, "target", msg.Target, "msg", msg.Message)
	default:
		slog.Error("unknown message arrived at deadletter", "msg", msg)
	}
}
