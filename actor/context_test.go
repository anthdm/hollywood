package actor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
