package actor

import (
	"fmt"
	"github.com/anthdm/hollywood/log"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestDeadLetterDefault tests the default deadletter handling.
// It will spawn a new actor, kill it, send a message to it and then check if the deadletter
// received the message.
func TestDeadLetterDefault(t *testing.T) {
	lh := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelDebug)
	e := NewEngine(EngineOptLogger(log.NewLogger("[engine]", lh)))
	a1 := e.Spawn(newTestActor, "a1")
	assert.NotNil(t, a1)
	dl := e.Registry.getByID("deadletter")
	assert.NotNil(t, dl)           // should be registered by default
	e.Poison(a1).Wait()            // poison the a1 actor
	e.Send(a1, testMessage{"bar"}) // should end up the deadletter queue
	time.Sleep(time.Millisecond)   // a flush would be nice here
	// flushes the deadletter queue, and returns the messages:
	resp, err := e.Request(dl.PID(), &DeadLetterFetch{Flush: true}, time.Millisecond*10).Result()
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	respDeadLetters, ok := resp.([]*DeadLetterEvent)
	assert.True(t, ok)                       // should be a slice of deadletter events
	assert.Equal(t, 1, len(respDeadLetters)) // should be one deadletter event
	ev, ok := respDeadLetters[0].Message.(testMessage)
	assert.True(t, ok) // should be a test message
	assert.Equal(t, "bar", ev.data)

}

// TestDeadLetterCustom tests the custom deadletter handling.
// It will spawn a new actor, kill it, send a message to it and then check if the deadletter
// received the message.
// It is using the custom deadletter receiver below.
func TestDeadLetterCustom(t *testing.T) {
	lh := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelDebug)
	e := NewEngine(
		EngineOptLogger(log.NewLogger("[engine]", lh)),
		EngineOptDeadletter(newCustomDeadLetter))
	a1 := e.Spawn(newTestActor, "a1")
	assert.NotNil(t, a1)
	dl := e.Registry.getByID("deadletter")
	assert.NotNil(t, dl) // should be registered by default
	// kill a1 actor.
	e.Poison(a1).Wait() // poison the a1 actor
	// should be in deadletter
	fmt.Println("==== sending message via a1 to deadletter ====")
	e.Send(a1, testMessage{"bar"})
	time.Sleep(time.Millisecond) // a flush would be nice here :-)
	resp, err := e.Request(dl.PID(), &DeadLetterFetch{Flush: true}, time.Millisecond*10).Result()
	assert.Nil(t, err)     // no error from the request
	assert.NotNil(t, resp) // we should get a response to our request
	respDeadLetters, ok := resp.([]*DeadLetterEvent)
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
func (t testActor) Receive(ctx *Context) {
	// do nothing
}

// customDeadLetter is a custom deadletter actor / receiver
type customDeadLetter struct {
	deadLetters []*DeadLetterEvent
}

func newCustomDeadLetter() Receiver {
	return &customDeadLetter{
		deadLetters: make([]*DeadLetterEvent, 0),
	}
}

// Receive implements the Receiver interface. This is a OK example of an actor that
// that deals with deadletters. It will store the deadletters in a slice.
func (c *customDeadLetter) Receive(ctx *Context) {
	switch ctx.Message().(type) {
	case *DeadLetterFlush:
		c.deadLetters = c.deadLetters[:0]
	case *DeadLetterFetch:
		ctx.Respond(c.deadLetters)
	case *DeadLetterEvent:
		slog.Warn("received deadletter event")
		msg, ok := ctx.Message().(*DeadLetterEvent)
		if !ok {
			slog.Error("failed to cast deadletter event")
			return
		}
		c.deadLetters = append(c.deadLetters, msg)
	}
}
