package actor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextSendRepeat(t *testing.T) {
	var (
		e  = NewEngine()
		wg = &sync.WaitGroup{}
		mu sync.Mutex
		sr SendRepeater
	)
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
	pidSeparator = ">"
	var (
		e           = NewEngine()
		wg          = sync.WaitGroup{}
		childfn     = func(c *Context) {}
		expectedPID = NewPID(LocalLookupAddr, "parent", "child")
	)

	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			pid := c.SpawnChildFunc(childfn, "child")
			assert.True(t, expectedPID.Equals(pid))
			wg.Done()
		case Stopped:
		}
	}, "parent")

	wg.Wait()
	pidSeparator = "/"
}

func TestChild(t *testing.T) {
	var (
		e  = NewEngine()
		wg = sync.WaitGroup{}
	)
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Initialized:
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithTags("1"))
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithTags("2"))
			c.SpawnChildFunc(func(_ *Context) {}, "child", WithTags("3"))
		case Started:
			assert.Equal(t, 3, len(c.Children()))
			wg.Done()
		}
	}, "foo", WithTags("bar", "baz"))
	wg.Wait()
}

func TestParent(t *testing.T) {
	var (
		e      = NewEngine()
		wg     = sync.WaitGroup{}
		parent = NewPID(LocalLookupAddr, "foo", "bar", "baz")
	)
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
	}, "foo", WithTags("bar", "baz"))

	wg.Wait()
}

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
		e  = NewEngine()
		wg = sync.WaitGroup{}
	)

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
	stopwg := &sync.WaitGroup{}
	e.Poison(pid, stopwg)
	stopwg.Wait()

	assert.Equal(t, e.deadLetter, e.Registry.get(NewPID("local", "child")))
	assert.Equal(t, e.deadLetter, e.Registry.get(pid))
}
