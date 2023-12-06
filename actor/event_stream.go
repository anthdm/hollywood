package actor

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
)

type EventStream struct {
	subs   map[*PID]bool
	dlsubs map[*PID]bool
}

func NewEventStream() Producer {
	return func() Receiver {
		return &EventStream{
			subs:   make(map[*PID]bool),
			dlsubs: make(map[*PID]bool),
		}
	}
}

// Receive for the event stream. All system-wide events are sent here.
// Some events are specially handled, such as EventSub, EventUnSub (for subscribing to events),
// DeadletterSub, DeadletterUnSub, for subscribing to DeadLetterEvent
func (e *EventStream) Receive(c *Context) {
	fmt.Printf("EventStream.Receive: %v\n", reflect.TypeOf(c.Message()))
	switch msg := c.Message().(type) {
	case EventSub:
		e.subs[msg.pid] = true
		fmt.Println("EventStream.Receive: EventSub")
	case EventUnsub:
		delete(e.subs, msg.pid)
		fmt.Println("EventStream.Receive: EventUnsub")
	case DeadletterSub:
		e.dlsubs[msg.pid] = true
	case DeadletterUnSub:
		delete(e.subs, msg.pid)
	case DeadLetterEvent:
		// to avoid a loop, check that the message isn't a deadletter.
		_, ok := msg.Message.(DeadLetterEvent)
		if ok {
			c.engine.BroadcastEvent(DeadLetterLoopEvent{})
			break
		}
		for sub := range e.dlsubs {
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
