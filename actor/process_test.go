package actor

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type triggerPanic struct {
	data int
}

func Test_DropsMessageAfterRetries(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)

	ch := make(chan any, 1)

	pid := e.SpawnFunc(func(c *Context) {
		fmt.Printf("Actor state: %T\n", c.Message())
		switch m := c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.pid)
		case triggerPanic:
			if m.data == 2 {
				panicWrapper()
			} else {
				fmt.Println("finally processed message", m.data)
			}
		case ActorUnprocessableMessageEvent:
			fmt.Printf("Got message type %T data:%d\n", m.Message, m.Message.(triggerPanic).data)
			ch <- m.Message
		}
	}, "kind", WithMaxRetries(1))

	e.Send(pid, triggerPanic{1})
	e.Send(pid, triggerPanic{2})
	e.Send(pid, triggerPanic{3})

	var messages []any
	select {
	case m := <-ch:
		messages = append(messages, m)
	case <-time.After(2 * time.Second):
		t.Error("timeout")
	}
	require.Len(t, messages, 1)
	require.IsType(t, triggerPanic{}, messages[0])
	require.Equal(t, triggerPanic{data: 2}, messages[0])
}

// Test_CleanTrace tests that the stack trace is cleaned up correctly and that the function
// which triggers the panic is at the top of the stack trace.
func Test_CleanTrace(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	require.NoError(t, err)
	ch := make(chan any)
	pid := e.SpawnFunc(func(c *Context) {
		fmt.Printf("Got message type %T\n", c.Message())
		switch m := c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.pid)
		case triggerPanic:
			if m.data != 10 {
				panicWrapper()
			} else {
				fmt.Println("finally processed all my messages", m.data)
			}
		case ActorRestartedEvent:
			// split the panic into lines:
			lines := bytes.Split(m.Stacktrace, []byte("\n"))
			// check that the second line is the panicWrapper function
			assert.True(t, bytes.Contains(lines[1], []byte("panicWrapper")))
			close(ch)
		}
	}, "foo", WithRetries(2), WithMaxRetries(2), WithMaxRestarts(1))
	e.Send(pid, triggerPanic{2})
	e.Send(pid, triggerPanic{3})
	e.Send(pid, triggerPanic{10})
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Error("test timed out")
		return
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
