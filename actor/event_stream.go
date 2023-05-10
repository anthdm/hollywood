package actor

import (
	"math"
	"math/rand"

	"github.com/anthdm/hollywood/log"
	"github.com/anthdm/hollywood/safemap"
)

type EventSub struct {
	id uint32
}

type EventStreamFunc func(event any)

type EventStream struct {
	subs *safemap.SafeMap[*EventSub, EventStreamFunc]
}

func NewEventStream() *EventStream {
	return &EventStream{
		subs: safemap.New[*EventSub, EventStreamFunc](),
	}
}

func (e *EventStream) Unsubscribe(sub *EventSub) {
	e.subs.Delete(sub)

	log.Tracew("[EVENTSTREAM] unsubscribe", log.M{
		"subs": e.subs.Len(),
		"id":   sub.id,
	})
}

func (e *EventStream) Subscribe(f EventStreamFunc) *EventSub {
	sub := &EventSub{
		id: uint32(rand.Intn(math.MaxUint32)),
	}
	e.subs.Set(sub, f)

	log.Tracew("[EVENTSTREAM] subscribe", log.M{
		"subs": e.subs.Len(),
		"id":   sub.id,
	})

	return sub
}

func (e *EventStream) Publish(msg any) {
	e.subs.ForEach(func(_ *EventSub, v EventStreamFunc) {
		go v(msg)
	})
}

func (e *EventStream) Len() int {
	return e.subs.Len()
}
