package actor

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// Test_CleanTrace tests that the stack trace is cleaned up correctly and that the function
// which triggers the panic is at the top of the stack trace.
func Test_CleanTrace(t *testing.T) {
	e, err := NewEngine(nil)
	require.NoError(t, err)
	type triggerPanic struct {
		data int
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
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
				wg.Done()
			} else {
				t.Error("stack trace does not contain panicWrapper at the right line")
				t.Fail()
			}
		}
	}, "foo", WithMaxRestarts(1))
	e.Send(pid, triggerPanic{1})
	wg.Wait()
}

func panicWrapper() {
	panic("foo")
}
