package actor

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeadLetterCustom tests the custom deadletter handling.
// It is using the custom deadletter receiver defined inline.
func TestDeadLetterCustom(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Initialized:
			c.engine.Subscribe(c.PID())
		case DeadLetterEvent:
			wg.Done()
		}
	}, "deadletter")
	e.SendLocal(invalidPid(), "bar", nil)
	wg.Wait()
}

// SafeBuffer is a threadsafe buffer, used for testing the that the deadletters are logged.
type SafeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *SafeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func invalidPid() *PID {
	return &PID{
		Address: "local",
		ID:      "squirrel",
	}
}
