package actor

import (
	"log/slog"
	"reflect"
)

// DeadLetterEvent is delivered to the deadletter actor when a message can't be delivered to it's recipient
type DeadLetterEvent struct {
	Target  *PID
	Message any
	Sender  *PID
}

type deadLetter struct {
	pid *PID
}

func newDeadLetter() Receiver {
	pid := NewPID(LocalLookupAddr, "deadletter")
	return &deadLetter{
		pid: pid,
	}
}

// Receive implements the Receiver interface, handling the deadletter messages.
// Any production system should provide a custom deadletter handler.
func (d *deadLetter) Receive(ctx *Context) {
	switch msg := ctx.Message().(type) {
	case Started:
		ctx.engine.BroadcastEvent(EventSub{pid: d.pid})
	case Stopped:
		ctx.engine.BroadcastEvent(EventUnsub{pid: d.pid})
	case Initialized:
	case DeadLetterEvent:
		slog.Warn("deadletter arrived", "msg-type", reflect.TypeOf(msg),
			"sender", msg.Sender, "target", msg.Target, "msg", msg.Message)
	default:
		slog.Error("unknown message arrived at deadletter", "msg", msg,
			"msg-type", reflect.TypeOf(msg))
	}
}
