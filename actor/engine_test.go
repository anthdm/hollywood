package actor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummy struct{}

func newDummy() Receiver {
	return &dummy{}
}

func (d *dummy) Receive(_ *Context) {}

func TestSpawn(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			pid := e.Spawn(newDummy, "dummy")
			e.Send(pid, 1)
			wg.Done()
		}()
	}

	wg.Wait()
	assert.Equal(t, 10, len(e.registry.procs))
}

func TestEventStream(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	subs := []*EventSub{}
	var mu sync.RWMutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			sub := e.EventStream.Subscribe(func(event any) {
				s, ok := event.(string)
				assert.True(t, ok)
				assert.Equal(t, "foo", s)
			})

			e.EventStream.Publish("foo")
			mu.Lock()
			subs = append(subs, sub)
			mu.Unlock()
			wg.Done()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 10, e.EventStream.Len())

	for _, sub := range subs {
		e.EventStream.Unsubscribe(sub)
	}

	assert.Equal(t, 0, e.EventStream.Len())
}
