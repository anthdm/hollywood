package actor

import (
	"github.com/anthdm/hollywood/log"
	"sync"
)

// TODO: The deadLetter is implemented as a plain Processer, but
// can actually be implemented as a Receiver. This is a good first issue.

type deadLetter struct {
	logger log.Logger
	pid    *PID
	msgs   []any
}

func newDeadLetter(logger log.Logger) Receiver {
	return &deadLetter{
		msgs:   make([]any, 0),
		logger: logger.SubLogger("[deadLetter]"),
		pid:    NewPID(LocalLookupAddr, "deadLetter"),
	}
}

func (d *deadLetter) Receive(ctx *Context) {
	switch msg := ctx.Message().(type) {
	case *DeadLetterEvent:
		d.msgs = append(d.msgs, msg)
	case *DeadLetterFlush:
		d.msgs = make([]any, 0)
	case *DeadLetterFetch:
		ctx.Respond(d.msgs)
		if msg.Flush {
			d.msgs = make([]any, 0)
		}
	}
}

func (d *deadLetter) PID() *PID                  { return d.pid }
func (d *deadLetter) Shutdown(_ *sync.WaitGroup) {}
func (d *deadLetter) Start()                     {}
func (d *deadLetter) Invoke([]Envelope)          {}
