package actor

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test_CleanTrace tests that the stack trace is cleaned up correctly and that the function
// which triggers the panic is at the top of the stack trace.
func Test_CleanTrace(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	type triggerPanic struct {
		data int
	}
	stopCh := make(chan struct{})
	pid := e.SpawnFunc(func(c *Context) {
		fmt.Printf("Got message type %T\n", c.Message())
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.pid)
		case triggerPanic:
			panicWrapper()
		case ActorRestartedEvent:
			m := c.Message().(ActorRestartedEvent)
			// split the panic into lines:
			lines := bytes.Split(m.Stacktrace, []byte("\n"))
			// check that the second line is the panicWrapper function:
			if bytes.Contains(lines[1], []byte("panicWrapper")) {
				fmt.Println("stack trace contains panicWrapper at the right line")
				stopCh <- struct{}{}
			}
		}
	}, "foo", WithMaxRestarts(1))
	e.Send(pid, triggerPanic{1})
	select {
	case <-stopCh:
		fmt.Println("test passed")
	case <-time.After(time.Second):
		t.Error("test timed out. stack trace likely did not contain panicWrapper at the right line")
	}
}

func panicWrapper() {
	panic("foo")
}

func Test_StartupMessages(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)

	type testMsg struct{}

	timeout := time.After(1 * time.Second)
	msgCh := make(chan any, 10) // used to check the msg order
	syncCh1 := make(chan struct{})
	syncCh2 := make(chan struct{})

	go func() {
		e.SpawnFunc(func(c *Context) {
			fmt.Printf("Got message type %T\n", c.Message())
			switch c.Message().(type) {
			case Initialized:
				syncCh1 <- struct{}{}
				// wait for testMsg to send
				select {
				case <-syncCh2:
				case <-timeout:
					t.Error("test timed out")
				}
			}
			msgCh <- c.Message()
		}, "foo", WithID("bar"))
	}()

	// wait for actor to initialize
	select {
	case <-syncCh1:
	case <-timeout:
		t.Error("test timed out")
		return
	}

	pid := e.Registry.GetPID("foo", "bar")
	e.Send(pid, testMsg{})
	syncCh2 <- struct{}{}

	// check that message order is as expected
	select {
	case msg := <-msgCh:
		_, ok := msg.(Initialized)
		require.True(t, ok)
	case <-timeout:
		t.Error("test timed out")
		return
	}

	select {
	case msg := <-msgCh:
		_, ok := msg.(Started)
		require.True(t, ok)
	case <-timeout:
		t.Error("test timed out")
		return
	}

	select {
	case msg := <-msgCh:
		_, ok := msg.(testMsg)
		require.True(t, ok)
	case <-timeout:
		t.Error("test timed out")
		return
	}
}

func Test_LifecyleMessages(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)

	type testMsg struct{}
	type stopMsg struct{}

	timeout := time.After(2 * time.Second)
	msgCh := make(chan any, 4)
	syncCh := make(chan any)

	go func() {
		e.SpawnFunc(func(c *Context) {
			fmt.Printf("Got message type %T\n", c.Message())
			switch msg := c.Message().(type) {
			case Initialized:
				syncCh <- msg
			case Started:
				syncCh <- msg
				c.Send(c.PID(), testMsg{})
			case Stopping:
				syncCh <- msg
			case Stopped:
				syncCh <- msg
			case testMsg:
				msgCh <- msg
				c.Send(c.PID(), stopMsg{})
			case stopMsg:
				c.engine.Poison(c.PID())
			}
		}, "foo", WithID("bar"))
	}()

	var lifecycle []any
	var messages []any
	// wait for actor to initialize
init:
	for {
		select {
		case msg := <-syncCh:
			lifecycle = append(lifecycle, msg)
			if _, ok := msg.(Stopped); ok {
				break init
			}
		case msg := <-msgCh:
			messages = append(messages, msg.(testMsg))
		case <-timeout:
			t.Error("test timed out")
			return
		}
	}
	require.Len(t, lifecycle, 4)
	require.Len(t, messages, 1)
	require.IsType(t, Initialized{}, lifecycle[0])
	require.IsType(t, Started{}, lifecycle[1])
	require.IsType(t, Stopping{}, lifecycle[2])
	require.IsType(t, Stopped{}, lifecycle[3])
	require.IsType(t, testMsg{}, messages[0])
}
