package actor

import (
	"context"
	"fmt"
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
		fmt.Println("EventStream.Receive: EventSub")
	case EventUnsub:
		delete(e.subs, msg.pid)
		fmt.Println("EventStream.Receive: EventUnsub")
	case DeadLetterEvent:
		// to avoid a loop, check that the message isn't a deadletter.
		_, ok := msg.Message.(DeadLetterEvent)
		if ok {
			c.engine.BroadcastEvent(DeadLetterLoopEvent{})
			break
		}
		if len(e.subs) == 0 {
			slog.Warn("deadletter arrived, but no subscribers to event stream",
				"sender", msg.Sender, "target", msg.Target, "msg", msg.Message)
			break
		}
		for sub := range e.subs {
			c.Forward(sub)
		}
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
