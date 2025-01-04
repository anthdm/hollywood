package actor

import (
	"context"
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

func TestSpawnWithContext(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	type key struct {
		key string
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx := context.WithValue(context.Background(), key{"foo"}, "bar")
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			assert.Equal(t, "bar", ctx.Value(key{"foo"}))
			wg.Done()
		}
	}, "test", WithContext(ctx))
	wg.Wait()
}

func TestRegistryGetPID(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	expectedPID1 := e.SpawnFunc(func(c *Context) {}, "foo", WithID("1"))
	expectedPID2 := e.SpawnFunc(func(c *Context) {}, "foo", WithID("2"))
	pid := e.Registry.GetPID("foo", "1")
	assert.True(t, pid.Equals(expectedPID1))
	pid = e.Registry.GetPID("foo", "2")
	assert.True(t, pid.Equals(expectedPID2))
}

func TestSendToNilPID(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	e.Send(nil, "foo")
}

func TestSendRepeat(t *testing.T) {
	var (
		wg = &sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)
	pid := e.Spawn(newTickReceiver(wg), "test")
	repeater := e.SendRepeat(pid, tick{}, time.Millisecond*2)
	wg.Wait()
	repeater.Stop()
}

func TestRestartsMaxRestarts(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	restarts := 2
	type payload struct {
		data int
	}
	pid := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case Started:
		case Stopped:
		case payload:
			if msg.data != 1 {
				panic("I failed to process this message")
			} else {
				fmt.Println("finally processed all my messages after borking.", msg.data)
			}
		}
	}, "foo", WithMaxRestarts(restarts))

	for i := 0; i < 2; i++ {
		e.Send(pid, payload{i})
	}
	<-e.Poison(pid).Done()
}

func TestProcessInitStartOrder(t *testing.T) {
	var (
		wg            = sync.WaitGroup{}
		started, init bool
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
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

func TestRestarts(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
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
				fmt.Println("finally processed all my messages after borking", msg.data)
				wg.Done()
			}
		}
	}, "foo", WithRestartDelay(time.Millisecond*10))

	e.Send(pid, payload{1})
	e.Send(pid, payload{2})
	e.Send(pid, payload{10})
	wg.Wait()
}

func TestSendWithSender(t *testing.T) {
	var (
		sender = NewPID("local", "sender")
		wg     = sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
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
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg := sync.WaitGroup{}

	pid := e.SpawnFunc(func(c *Context) {
		msg := c.Message()
		if msg == nil {
			fmt.Println("should never happen")
		}
	}, "test")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		e.Send(pid, []byte("f"))
		wg.Done()
	}
	wg.Wait()
}

func TestSpawn(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			tag := strconv.Itoa(i)
			pid := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {
			}), "dummy", WithID(tag))
			e.Send(pid, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestSpawnDuplicateKind(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	pid1 := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {}), "dummy")
	e.Send(pid1, 1)
	pid2 := e.Spawn(NewTestProducer(t, func(t *testing.T, ctx *Context) {}), "dummy")
	e.Send(pid2, 2)
	wg.Wait()
}

func TestSpawnDuplicateId(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	var startsCount int32 = 0
	receiveFunc := func(t *testing.T, ctx *Context) {
		switch ctx.Message().(type) {
		case Initialized:
			atomic.AddInt32(&startsCount, 1)
		}
	}
	e.Spawn(NewTestProducer(t, receiveFunc), "dummy", WithID("1"))
	e.Spawn(NewTestProducer(t, receiveFunc), "dummy", WithID("1"))
	assert.Equal(t, int32(1), startsCount) // should only spawn one actor
}

func TestStopWaitContextDone(t *testing.T) {
	var (
		wg = sync.WaitGroup{}
		x  = int32(0)
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
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

	<-e.Stop(pid).Done()
	assert.Equal(t, int32(1), atomic.LoadInt32(&x))
}

func TestStop(t *testing.T) {
	var (
		wg = sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		tag := strconv.Itoa(i)
		pid := e.SpawnFunc(func(c *Context) {
			switch c.Message().(type) {
			case Started:
				wg.Done()
			case Stopped:
			}
		}, "foo", WithID(tag))

		wg.Wait()
		<-e.Stop(pid).Done()
		// When a process is poisoned it should be removed from the registry.
		// Hence, we should get nil when looking it up in the registry.
		assert.Nil(t, e.Registry.get(pid))
	}
}

func TestPoisonWaitGroup(t *testing.T) {
	var (
		wg = &sync.WaitGroup{}
		x  = int32(0)
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
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

	<-e.Poison(pid).Done()
	assert.Equal(t, int32(1), atomic.LoadInt32(&x))

	// validate poisoning non exiting pid does not deadlock
	e.Poison(NewPID(LocalLookupAddr, "non-existing"))
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Millisecond):
		t.Error("poison waitGroup deadlocked")
	}
}

func TestPoison(t *testing.T) {
	var (
		wg = sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		tag := strconv.Itoa(i)
		pid := e.SpawnFunc(func(c *Context) {
			switch c.Message().(type) {
			case Started:
				wg.Done()
			case Stopped:
			}
		}, "foo", WithID(tag))

		wg.Wait()
		<-e.Poison(pid).Done()
		// When a process is poisoned it should be removed from the registry.
		// Hence, we should get NIL when we try to get it.
		assert.Nil(t, e.Registry.get(pid))
	}
}

func TestRequestResponse(t *testing.T) {
	type responseEvent struct {
		d time.Duration
	}
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)
	a := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case responseEvent:
			d := c.Message().(responseEvent).d
			time.Sleep(d)
			c.Respond("foo")
		}
	}, "actor_a")
	t.Run("should timeout", func(t *testing.T) {
		// a task with a 1us timeout which takes 20ms to complete, should always time out.
		resp := e.Request(a, responseEvent{d: time.Millisecond * 20}, 1*time.Microsecond)
		_, err := resp.Result()
		assert.Error(t, err)
		assert.Nil(t, e.Registry.get(resp.pid))

	})
	t.Run("should not timeout", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			resp := e.Request(a, responseEvent{d: time.Microsecond * 1}, time.Millisecond*800)
			res, err := resp.Result()
			assert.NoError(t, err)
			assert.Equal(t, "foo", res)
			assert.Nil(t, e.Registry.get(resp.pid))
		}
	})
}

func TestPoisonPillPrivate(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	successCh := make(chan struct{}, 1)
	failCh := make(chan struct{}, 1)
	pid := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Stopped:
			successCh <- struct{}{}
		case poisonPill:
			failCh <- struct{}{}
			time.Sleep(time.Millisecond)
		}
	}, "victim")
	<-e.Poison(pid).Done()
	assert.Nil(t, e.Registry.get(pid))
	select {
	case <-failCh:
		t.Fatal("poison pill seen")
	case <-successCh:
		return // actor was stopped without seeing a poison pill
	case <-time.After(time.Millisecond * 20):
		t.Fatal("timeout")
	}
}

// 45.84 ns/op 25 B/op => 13th Gen Intel(R) Core(TM) i9-13900KF
func BenchmarkSendMessageLocal(b *testing.B) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(b, err)
	pid := e.SpawnFunc(func(_ *Context) {}, "bench", WithInboxSize(128))

	b.ResetTimer()
	b.Run("send_message_local", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			e.Send(pid, pid)
		}
	})
}

func BenchmarkSendWithSenderMessageLocal(b *testing.B) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(b, err)
	p := NewTestProducer(nil, func(_ *testing.T, _ *Context) {})
	pid := e.Spawn(p, "bench", WithInboxSize(1024*8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.SendWithSender(pid, pid, pid)
	}
}

type TestReceiveFunc func(*testing.T, *Context)

type TestReceiver struct {
	OnReceive TestReceiveFunc
	t         *testing.T
}

func NewTestProducer(t *testing.T, f TestReceiveFunc) Producer {
	return func() Receiver {
		return &TestReceiver{
			OnReceive: f,
			t:         t,
		}
	}
}

func (r *TestReceiver) Receive(ctx *Context) {
	r.OnReceive(r.t, ctx)
}

func TestMultipleStops(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	for i := 0; i < 1000; i++ {
		done := make(chan struct{})
		pid := e.SpawnFunc(func(ctx *Context) {
			switch ctx.Message().(type) {
			case Stopped:
				close(done)
			}
		}, "test")
		for j := 0; j < 10; j++ {
			e.Stop(pid)
		}
		<-done
	}
}
