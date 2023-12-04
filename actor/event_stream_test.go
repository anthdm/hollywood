package actor

import (
	fmt "fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

type CustomEvent struct {
	msg string
}

func TestEventStreamLocal(t *testing.T) {
	e, err := NewEngine()
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

}

func TestEventStreamActorStoppedEvent(t *testing.T) {

}
