package actor

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

func (e *EventStream) Receive(c *Context) {
	switch msg := c.Message().(type) {
	case EventSub:
		e.subs[msg.pid] = true
	case EventUnsub:
		delete(e.subs, msg.pid)
	default:
		for sub := range e.subs {
			c.Forward(sub)
		}
	}
}
