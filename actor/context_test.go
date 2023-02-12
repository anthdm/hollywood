package actor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLocalPID(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		if _, ok := c.Message().(Started); ok {
			pid := c.GetLocalPID("foo", "bar", "baz")
			require.True(t, pid.Equals(c.PID()))
			wg.Done()
		}
	}, "foo", "bar", "baz")
	wg.Wait()
}

func TestSpawnChild(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	pid := e.SpawnFunc(func(ctx *Context) {
		if _, ok := ctx.Message().(Started); ok {
			wg.Add(1)
			ctx.SpawnChildFunc(func(cc *Context) {
				switch cc.Message().(type) {
				case Stopped:
					wg.Done()
				}
			}, "child")
		}
	}, "parent")
	e.Poison(pid)
	wg.Wait()
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "child")))
	assert.Equal(t, e.deadLetter, e.registry.get(pid))
}
