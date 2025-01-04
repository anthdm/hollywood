package actor

import (
	fmt "fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomEvent struct {
	msg string
}

func TestEventStreamLocal(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(2)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case CustomEvent:
			fmt.Println("actor a received event")
			wg.Done()
		}
	}, "actor_a")

	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case CustomEvent:
			fmt.Println("actor b received event")
			wg.Done()
		}
	}, "actor_b")
	e.BroadcastEvent(CustomEvent{msg: "foo"})
	// make sure both actors have received the event.
	// If so, the test has passed.
	wg.Wait()
}

func TestEventStreamActorStartedEvent(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	wg := sync.WaitGroup{}

	wg.Add(1)
	pidb := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case ActorStartedEvent:
			assert.Equal(t, msg.PID.ID, "a/1")
			wg.Done()
		}
	}, "b")
	e.Subscribe(pidb)

	e.SpawnFunc(func(c *Context) {}, "a", WithID("1"))
	wg.Wait()
}

func TestEventStreamActorStoppedEvent(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	wg := sync.WaitGroup{}

	wg.Add(1)
	a := e.SpawnFunc(func(c *Context) {}, "a", WithID("1"))
	pidb := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case ActorStoppedEvent:
			assert.Equal(t, msg.PID.ID, "a/1")
			wg.Done()
		}
	}, "b")

	e.Subscribe(pidb)
	<-e.Poison(a).Done()

	wg.Wait()
}
