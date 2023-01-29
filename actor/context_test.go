package actor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func _TestSpawnChild(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	wg.Add(1)

	e.SpawnFunc(func(ctx *Context) {
		switch ctx.Message().(type) {
		case Started:
			cpid := ctx.SpawnChildFunc(func(cctx *Context) {
				switch cctx.Message().(type) {
				case Started:
					// go e.Poison(ctx.pid)
					fmt.Println("child started")
				case Stopped:
					// wg.Done()
					fmt.Println("wg called")
					fmt.Println("child stopped")
				}
			}, "test_child")

			go e.Poison(ctx.pid)

			fmt.Println("parent started with child", cpid)
		case Stopped:
			fmt.Println("parent stopped")
		}
	}, "test")

	wg.Wait()
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "test_child")))
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "test")))
}
