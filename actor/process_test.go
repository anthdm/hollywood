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
