package actor

import (
	"github.com/anthdm/hollywood/log"
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
// Todo: this will grow and grow. Maybe we want a limit on this?
func (d *deadLetter) Receive(ctx *Context) {
	switch msg := ctx.Message().(type) {
	case Started:
		// intialize logger on deadletter startup. this should be sanity checked
		d.logger = ctx.Engine().logger.SubLogger("[deadletter]")
		d.logger.Debugw("default deadletter actor started")
	case Stopped:
		d.logger.Debugw("default deadletter actor stopped")
	case Initialized:
		d.logger.Debugw("default deadletter actor initialized")
	case *DeadLetterFlush:
		d.logger.Debugw("deadletter queue flushed", "msgs", len(d.msgs), "sender", ctx.Sender())
		d.msgs = make([]*DeadLetterEvent, 0)
	case *DeadLetterFetch:
		d.logger.Debugw("deadletter fetch", "msgs", len(d.msgs), "sender", ctx.Sender(), "flush", msg.Flush)
		ctx.Respond(d.msgs) // this is a sync request.
		if msg.Flush {
			d.msgs = d.msgs[:0]
		}
	case *DeadLetterEvent:
		d.logger.Warnw("deadletter arrived", "msg", msg, "sender", ctx.Sender())
		d.msgs = append(d.msgs, msg)
	default:
		d.logger.Errorw("unknown message arrived", "msg", msg)
	}
}
