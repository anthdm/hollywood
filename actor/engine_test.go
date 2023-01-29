package actor

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendWithSender(t *testing.T) {
	e := NewEngine()
	sender := NewPID("local", "foo")
	wg := sync.WaitGroup{}
	wg.Add(1)
	pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
		if _, ok := ctx.Message().(string); ok {
			assert.NotNil(t, ctx.Sender())
			assert.Equal(t, sender, ctx.Sender())
			wg.Done()
		}
	}), "test")
	e.SendWithSender(pid, "data", sender)
	wg.Wait()
}

func TestSendMsgRaceCon(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
		msg := ctx.Message()
		if msg == nil {
			fmt.Println("should never happen")
		}
	}), "test")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			e.Send(pid, []byte("f"))
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestSpawn(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			tag := strconv.Itoa(i)
			pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
			}), "dummy", tag)
			e.Send(pid, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestPoison(t *testing.T) {
	e := NewEngine()

	for i := 0; i < 4; i++ {
		tag := strconv.Itoa(i)
		pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {

		}), "dummy", tag)
		e.Poison(pid)
		// When a process is poisoned it should be removed from the registry.
		// Hence, we should get the dead letter process here.
		assert.Equal(t, e.deadLetter, e.registry.get(pid))
	}
}

func TestRequestResponse(t *testing.T) {
	e := NewEngine()
	pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
		if msg, ok := ctx.Message().(string); ok {
			assert.Equal(t, "foo", msg)
			ctx.Respond("bar")
		}
	}), "dummy")
	resp := e.Request(pid, "foo", time.Millisecond)
	res, err := resp.Result()
	assert.Nil(t, err)
	assert.Equal(t, "bar", res)
	// Response PID should be the dead letter PID. This is because
	// the actual response process that will handle this RPC
	// is deregistered. Test that its actually cleaned up.
	assert.Equal(t, e.deadLetter, e.registry.get(resp.pid))
}

func BenchmarkSendMessageLocal(b *testing.B) {
	e := NewEngine()
	p := NewTestProducer(nil, func(_ *testing.T, _ *Context) {})

	pid := e.SpawnOpts(Opts{
		Producer:  p,
		InboxSize: 10000,
		Name:      "bench",
	})

	for i := 0; i < b.N; i++ {
		e.Send(pid, pid)
	}
}

func BenchmarkSendWithSenderMessageLocal(b *testing.B) {
	e := NewEngine()
	p := NewTestProducer(nil, func(_ *testing.T, _ *Context) {})

	pid := e.SpawnOpts(Opts{
		Producer:  p,
		InboxSize: 10000,
		Name:      "bench",
	})

	for i := 0; i < b.N; i++ {
		e.SendWithSender(pid, pid, pid)
	}
}
