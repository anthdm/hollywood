package actor

// EventSub
type EventSub struct {
	id  uint32 // backwards compat for now
	pid *PID
}

type EventUnsub struct {
	pid *PID
}

type Event struct {
	subs map[*PID]bool
}

func NewEvent() Producer {
	return func() Receiver {
		return &Event{
			subs: make(map[*PID]bool),
		}
	}
}

func (e *Event) Receive(c *Context) {
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
