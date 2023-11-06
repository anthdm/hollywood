package actor

import (
	"math"
	"math/rand"
	"sync"

	"github.com/anthdm/hollywood/log"
)

type EventSub struct {
	id uint32
}

type EventStreamFunc func(event any)

type EventStream struct {
	mu     sync.RWMutex
	subs   map[*EventSub]EventStreamFunc
	logger log.Logger
}

func NewEventStream(l log.Logger) *EventStream {
	return &EventStream{
		subs:   make(map[*EventSub]EventStreamFunc),
		logger: l.SubLogger("[eventStream]"),
	}
}

func (e *EventStream) Unsubscribe(sub *EventSub) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.subs, sub)

	e.logger.Debugw("unsubscribe",
		"subs", len(e.subs),
		"id", sub.id,
	)
}

func (e *EventStream) Subscribe(f EventStreamFunc) *EventSub {
	e.mu.Lock()
	defer e.mu.Unlock()

	sub := &EventSub{
		id: uint32(rand.Intn(math.MaxUint32)),
	}
	e.subs[sub] = f

	e.logger.Debugw("subscribe",
		"subs", len(e.subs),
		"id", sub.id,
	)

	return sub
}

func (e *EventStream) Publish(msg any) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, f := range e.subs {
		go f(msg)
	}
}

func (e *EventStream) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.subs)
}
