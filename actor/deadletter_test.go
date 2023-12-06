package actor

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// TestDeadLetterDefault tests the default deadletter handling.
// It will spawn a new actor, kill it, send a message to it and then check if there is a message
// logged to the default logger
func TestDeadLetterDefault(t *testing.T) {
	logBuffer := SafeBuffer{}
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	e, err := NewEngine()
	assert.NoError(t, err)
	e.Send(invalidPid(), "bar")  // should end up the deadletter queue
	time.Sleep(time.Millisecond) // wait for the deadletter to be processed
	// check the log buffer for the deadletter
	assert.Contains(t, logBuffer.String(), "deadletter arrived")
}

// TestDeadLetterCustom tests the custom deadletter handling.
// It is using the custom deadletter receiver defined inline.
func TestDeadLetterCustom(t *testing.T) {
	e, err := NewEngine()
	assert.NoError(t, err)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Initialized:
			c.engine.BroadcastEvent(EventSub{c.pid})
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
