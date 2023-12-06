package actor

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDeadLetterDefault tests the default deadletter handling.
// It will spawn a new actor, kill it, send a message to it and then check if the deadletter
// received the message.
func TestDeadLetterDefault(t *testing.T) {
	logBuffer := SafeBuffer{}
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	e, err := NewEngine()
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	a1 := e.Spawn(newTestActor, "a1")
	assert.NotNil(t, a1)
	dl := e.Registry.getByID("deadletter")
	assert.NotNil(t, dl)               // should be registered by default
	e.Poison(a1).Wait()                // poison the a1 actor
	e.Send(a1, testMessage{"bar"})     // should end up the deadletter queue
	time.Sleep(time.Millisecond * 100) // a flush would be nice here

	// check the log buffer for the deadletter
	logStr := logBuffer.String()
	fmt.Println(logStr)
	assert.Contains(t, logStr, "deadletter arrived")

}

// TestDeadLetterCustom tests the custom deadletter handling.
// It will spawn a new actor, kill it, send a message to it and then check if the deadletter
// received the message.
// It is using the custom deadletter receiver below.
func TestDeadLetterCustom(t *testing.T) {
	debuglogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(debuglogger)
	e, err := NewEngine(
		EngineOptDeadletter(newCustomDeadLetter))
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	a1 := e.Spawn(newTestActor, "a1")
	assert.NotNil(t, a1)
	dl := e.Registry.getByID("deadletter")
	assert.NotNil(t, dl)
	es := e.Registry.getByID("eventstream")
	assert.NotNil(t, es)

	// kill a1 actor.
	e.Poison(a1).Wait() // poison the a1 actor
	fmt.Println("==== sending message via a1 to deadletter ====")
	e.Send(a1, testMessage{"bar"})
	time.Sleep(time.Millisecond * 100) // a flush would be nice here :-)
	resp, err := e.Request(dl.PID(), customDeadLetterFetch{flush: true}, time.Millisecond*10).Result()
	assert.Nil(t, err)     // no error from the request
	assert.NotNil(t, resp) // we should get a response to our request
	respDeadLetters, ok := resp.([]DeadLetterEvent)
	assert.True(t, ok)                       // got a slice of deadletter events
	assert.Equal(t, 1, len(respDeadLetters)) // one deadletter event
	ev, ok := respDeadLetters[0].Message.(testMessage)
	assert.True(t, ok) // should be our test message
	assert.Equal(t, "bar", ev.data)
}

type testActor struct{}
type testMessage struct {
	data string
}

func newTestActor() Receiver {
	return testActor{}
}
func (t testActor) Receive(_ *Context) {
	// do nothing
}

type customDeadLetterFetch struct{ flush bool }

// customDeadLetter is a custom deadletter actor / receiver
type customDeadLetter struct {
	deadLetters []DeadLetterEvent
}

func newCustomDeadLetter() Receiver {
	return &customDeadLetter{
		deadLetters: make([]DeadLetterEvent, 0),
	}
}

// Receive implements the Receiver interface. This is a OK example of an actor that
// that deals with deadletters. It will store the deadletters in a slice.
func (c *customDeadLetter) Receive(ctx *Context) {
	es := ctx.engine.Registry.getByID("eventstream")
	if es == nil {
		slog.Error("custom deadletter; no eventstream found")
	}
	switch ctx.Message().(type) {
	case Started:
		slog.Debug("custom deadletter starting", "action", "subscribing")
		ctx.engine.BroadcastEvent(DeadletterSub{pid: ctx.pid})
		time.Sleep(time.Millisecond * 10)
	case Stopped:
		slog.Debug("custom deadletter stopping", "action", "unsubscribing")
		ctx.engine.BroadcastEvent(DeadletterUnSub{pid: ctx.pid})
	case customDeadLetterFetch:
		flush := ctx.Message().(customDeadLetterFetch).flush
		slog.Debug("custom deadletter; received fetch request",
			"flush", flush,
			"messages", len(c.deadLetters))
		ctx.Respond(c.deadLetters)
		if ctx.Message().(customDeadLetterFetch).flush {
			c.deadLetters = c.deadLetters[:0]
		}
	case DeadLetterEvent:
		slog.Warn("custom deadletter; received deadletter event")
		msg, ok := ctx.Message().(DeadLetterEvent)
		if !ok {
			slog.Error("should never happen. brain damaged.")
			return
		}
		c.deadLetters = append(c.deadLetters, msg)
	default:
		slog.Error("custom deadletter; received unknown message",
			"msg", ctx.Message(), "msg-type", reflect.TypeOf(ctx.Message()))
	}
}

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

// Usage in goroutines...
