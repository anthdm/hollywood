package actor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDuplicateIdEvent(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	monitor := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case ActorDuplicateIdEvent:
			wg.Done()
		}
	}, "monitor")
	e.SpawnFunc(func(c *Context) {}, "actor_a", WithID("1"))
	e.SpawnFunc(func(c *Context) {}, "actor_a", WithID("1"))
	wg.Wait()
	<-e.Poison(monitor).Done()
}
