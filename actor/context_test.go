package actor

import (
	fmt "fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChildEventNoRaceCondition(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.Nil(t, err)

	parentPID := e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			child := c.SpawnChildFunc(func(childctx *Context) {
			}, "child")
			c.engine.Subscribe(child)
		}
	}, "parent")
	<-e.Poison(parentPID).Done()
}

func TestContextSendRepeat(t *testing.T) {
	var (
		wg = &sync.WaitGroup{}
		mu sync.Mutex
		sr SendRepeater
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)

	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			mu.Lock()
			sr = c.SendRepeat(c.PID(), "foo", time.Millisecond*10)
			mu.Unlock()
		case string:
			mu.Lock()
			sr.Stop()
			mu.Unlock()
			assert.Equal(t, c.Sender(), c.PID())
			wg.Done()
		}
	}, "test")
	wg.Wait()
}

func TestSpawnChildPID(t *testing.T) {
	var (
		wg          = sync.WaitGroup{}
		childfn     = func(c *Context) {}
		expectedPID = NewPID(LocalLookupAddr, "parent/1/child/1")
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			pid := c.SpawnChildFunc(childfn, "child", WithID("1"))
			assert.True(t, expectedPID.Equals(pid))
			wg.Done()
		case Stopped:
		}
	}, "parent", WithID("1"))

	wg.Wait()
}

func TestChild(t *testing.T) {
	var (
		wg = sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Initialized:
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithID("1"))
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithID("2"))
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithID("3"))
		case Started:
			assert.Equal(t, 3, len(c.Children()))
			childPid := c.Children()[0]
			fmt.Println("sending poison pill to ", childPid)
			<-c.Engine().Stop(childPid).Done()
			assert.Equal(t, 2, len(c.Children()))
			wg.Done()
		}
	}, "foo", WithID("bar/baz"))
	wg.Wait()
}

func TestParent(t *testing.T) {
	var (
		wg     = sync.WaitGroup{}
		parent = NewPID(LocalLookupAddr, "foo/bar/baz")
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)

	childfn := func(c *Context) {
		switch c.Message().(type) {
		case Started:
			assert.True(t, c.Parent().Equals(parent))
			assert.True(t, len(c.Children()) == 0)
			wg.Done()
		}
	}

	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.SpawnChildFunc(childfn, "child")
		}
	}, "foo", WithID("bar/baz"))

	wg.Wait()
}

func TestGetPID(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		if _, ok := c.Message().(Started); ok {
			pid := c.GetPID("foo/bar")
			require.True(t, pid.Equals(c.PID()))
			wg.Done()
		}
	}, "foo", WithID("bar"))

	wg.Wait()
}

func TestSpawnChild(t *testing.T) {
	var (
		wg = sync.WaitGroup{}
	)
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	wg.Add(1)
	childFunc := func(c *Context) {
		switch c.Message().(type) {
		case Stopped:
		}
	}

	pid := e.SpawnFunc(func(ctx *Context) {
		switch ctx.Message().(type) {
		case Started:
			ctx.SpawnChildFunc(childFunc, "child", WithMaxRestarts(0))
			wg.Done()
		}
	}, "parent", WithMaxRestarts(0))

	wg.Wait()
	<-e.Poison(pid).Done()

	assert.Nil(t, e.Registry.get(NewPID("local", "child")))
	assert.Nil(t, e.Registry.get(pid))
}
