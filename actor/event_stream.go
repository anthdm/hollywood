package actor

import (
	"context"
	"log/slog"
)

// EventSub is the message that will be send to subscribe to the event stream.
type EventSub struct {
	pid *PID
}

// EventUnSub is the message that will be send to unsubscribe from the event stream.
type EventUnsub struct {
	pid *PID
}

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
		logMsg, ok := c.Message().(EventLogger)
		if ok {
			level, msg, attr := logMsg.Log()
			slog.Log(context.Background(), level, msg, attr...)
		}
		for sub := range e.subs {
			c.Forward(sub)
		}
	}
}
