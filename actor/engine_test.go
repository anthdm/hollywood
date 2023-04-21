package actor

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tick struct{}
type tickReceiver struct {
	ticks int
	wg    *sync.WaitGroup
}

func (r *tickReceiver) Receive(c *Context) {
	switch c.Message().(type) {
	case tick:
		r.ticks++
		if r.ticks == 10 {
			r.wg.Done()
		}
	}
}

func newTickReceiver(wg *sync.WaitGroup) Producer {
	return func() Receiver {
		return &tickReceiver{
			wg: wg,
		}
	}
}

func TestSendRepeat(t *testing.T) {
	var (
		e  = NewEngine()
		wg = &sync.WaitGroup{}
	)
	wg.Add(1)
	pid := e.Spawn(newTickReceiver(wg), "test")
	repeater := e.SendRepeat(pid, tick{}, time.Millisecond*2)
	wg.Wait()
	repeater.Stop()
}

func TestRestarts(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	type payload struct {
		data int
	}

	wg.Add(1)
	pid := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case Started:
		case Stopped:
			fmt.Println("stopped!")
		case payload:
			if msg.data != 10 {
				panic("I failed to process this message")
			} else {
				fmt.Println("finally processed all my messsages after borking.", msg.data)
				wg.Done()
			}
		}
	}, "foo", WithRestartDelay(time.Millisecond*10))

	e.Send(pid, payload{1})
	e.Send(pid, payload{2})
	e.Send(pid, payload{10})
	wg.Wait()
}

func TestProcessInitStartOrder(t *testing.T) {
	var (
		e             = NewEngine()
		wg            = sync.WaitGroup{}
		started, init bool
	)
	pid := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Initialized:
			fmt.Println("init")
			wg.Add(1)
			init = true
		case Started:
			fmt.Println("start")
			require.True(t, init)
			started = true
		case int:
			fmt.Println("msg")
			require.True(t, started)
			wg.Done()
		}
	}, "tst")
	e.Send(pid, 1)
	wg.Wait()
}

func TestSendWithSender(t *testing.T) {
	var (
		e      = NewEngine()
		sender = NewPID("local", "sender")
		wg     = sync.WaitGroup{}
	)
	wg.Add(1)

	pid := e.SpawnFunc(func(c *Context) {
		if _, ok := c.Message().(string); ok {
			assert.NotNil(t, c.Sender())
			assert.Equal(t, sender, c.Sender())
			wg.Done()
		}
	}, "test")
	e.SendWithSender(pid, "data", sender)
	wg.Wait()
}

func TestSendMsgRaceCon(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}

	pid := e.SpawnFunc(func(c *Context) {
		msg := c.Message()
		if msg == nil {
			fmt.Println("should never happen")
		}
	}, "test")

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
			}), "dummy", WithTags(tag))
			e.Send(pid, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestPoisonWaitGroup(t *testing.T) {
	var (
		e  = NewEngine()
		wg = sync.WaitGroup{}
		x  = int32(0)
	)
	wg.Add(1)

	pid := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			wg.Done()
		case Stopped:
			atomic.AddInt32(&x, 1)
		}
	}, "foo")
	wg.Wait()

	pwg := &sync.WaitGroup{}
	e.Poison(pid, pwg)
	pwg.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&x))
}

func TestPoison(t *testing.T) {
	var (
		e  = NewEngine()
		wg = sync.WaitGroup{}
	)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		tag := strconv.Itoa(i)
		pid := e.SpawnFunc(func(c *Context) {
			switch c.Message().(type) {
			case Started:
				wg.Done()
			case Stopped:
			}
		}, "foo", WithTags(tag))

		wg.Wait()
		stopwg := &sync.WaitGroup{}
		e.Poison(pid, stopwg)
		stopwg.Wait()
		// When a process is poisoned it should be removed from the registry.
		// Hence, we should get the dead letter process here.
		assert.Equal(t, e.deadLetter, e.Registry.get(pid))
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
	assert.Equal(t, e.deadLetter, e.Registry.get(resp.pid))
}

// 56 ns/op
func BenchmarkSendMessageLocal(b *testing.B) {
	e := NewEngine()
	p := NewTestProducer(nil, func(_ *testing.T, _ *Context) {})
	pid := e.Spawn(p, "bench", WithInboxSize(1024*8))

	b.ResetTimer()
	b.Run("send_message_local", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			e.Send(pid, pid)
		}
	})
}

func BenchmarkSendWithSenderMessageLocal(b *testing.B) {
	e := NewEngine()
	p := NewTestProducer(nil, func(_ *testing.T, _ *Context) {})
	pid := e.Spawn(p, "bench", WithInboxSize(1024*8))

	for i := 0; i < b.N; i++ {
		e.SendWithSender(pid, pid, pid)
	}
}
