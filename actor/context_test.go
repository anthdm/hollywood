package actor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPID(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		if _, ok := c.Message().(Started); ok {
			pid := c.GetPID("foo", "bar", "baz")
			require.True(t, pid.Equals(c.PID()))
			wg.Done()
		}
	}, "foo", WithTags("bar", "baz"))
	wg.Wait()
}

func TestSpawnChild(t *testing.T) {
	var (
		e      = NewEngine()
		wg     = sync.WaitGroup{}
		stopwg = sync.WaitGroup{}
	)

	wg.Add(1)
	stopwg.Add(1)
	pid := e.SpawnFunc(func(ctx *Context) {
		if _, ok := ctx.Message().(Started); ok {
			wg.Done()
			ctx.SpawnChildFunc(func(cc *Context) {
				switch cc.Message().(type) {
				case Stopped:
					stopwg.Done()
				}
			}, "child")
		}
	}, "parent")

	wg.Wait()
	e.Poison(pid)

	stopwg.Wait()
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "child")))
	assert.Equal(t, e.deadLetter, e.registry.get(pid))
}
