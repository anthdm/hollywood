package actor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSpawnChild(t *testing.T) {
	e := NewEngine()
	pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
		switch ctx.Message().(type) {
		case Started:
			var cpid *PID
			cpid = ctx.SpawnChild(NewTestProducer(t, func(_ *testing.T, _ *Context) {
				switch ctx.Message().(type) {
				case Started:
				case Stopped:
					assert.Equal(t, cpid, ctx.GetChild("test_child"))
				}
			}), "test_child")
		case Stopped:
		}
	}), "test")
	e.Poison(pid)
	time.Sleep(time.Millisecond)
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "test_child")))
	assert.Equal(t, e.deadLetter, e.registry.get(NewPID("local", "test")))
}
