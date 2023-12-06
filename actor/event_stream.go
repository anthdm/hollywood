package actor

import (
	"context"
	"log/slog"
)

type EventStream struct {
	subs map[*PID]bool
}

func NewEventStream() Producer {
	return func() Receiver {
		return &EventStream{
			subs: make(map[*PID]bool),
		}
	}
}

// Receive for the event stream. All system-wide events are sent here.
// Some events are specially handled, such as EventSub, EventUnSub (for subscribing to events),
// DeadletterSub, DeadletterUnSub, for subscribing to DeadLetterEvent
func (e *EventStream) Receive(c *Context) {
	switch msg := c.Message().(type) {
	case EventSub:
		e.subs[msg.pid] = true
	case EventUnsub:
		delete(e.subs, msg.pid)
	default:
		// check if we should log the event, if so, log it with the relevant level, message and attributes
		logMsg, ok := c.Message().(eventLog)
		if ok {
			level, msg, attr := logMsg.log()
			slog.Log(context.Background(), level, msg, attr...)
		}
		for sub := range e.subs {
			c.Forward(sub)
		}
	}
}
