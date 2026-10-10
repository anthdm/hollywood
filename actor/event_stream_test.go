package actor

import (
	fmt "fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type CustomEvent struct {
	msg string
}

func TestEventStreamLocal(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(2)
	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case CustomEvent:
			fmt.Println("actor a received event")
			wg.Done()
		}
	}, "actor_a")

	e.SpawnFunc(func(c *Context) {
		switch c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case CustomEvent:
			fmt.Println("actor b received event")
			wg.Done()
		}
	}, "actor_b")
	e.BroadcastEvent(CustomEvent{msg: "foo"})
	// make sure both actors have received the event.
	// If so, the test has passed.
	wg.Wait()
}

func TestEventStreamActorStartedEvent(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	wg := sync.WaitGroup{}

	wg.Add(1)
	pidb := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case ActorStartedEvent:
			assert.Equal(t, msg.PID.ID, "a/1")
			wg.Done()
		}
	}, "b")
	e.Subscribe(pidb)

	e.SpawnFunc(func(c *Context) {}, "a", WithID("1"))
	wg.Wait()
}

func TestEventStreamActorStoppedEvent(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	wg := sync.WaitGroup{}

	wg.Add(1)
	a := e.SpawnFunc(func(c *Context) {}, "a", WithID("1"))
	pidb := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case ActorStoppedEvent:
			assert.Equal(t, msg.PID.ID, "a/1")
			wg.Done()
		}
	}, "b")

	e.Subscribe(pidb)
	<-e.Poison(a).Done()

	wg.Wait()
}

type boundPing struct{ n int }

// boundMark is an ordering token. final is false for the flush and true
// for the sentinel.
type boundMark struct{ final bool }

func TestPoisonedSubscriberDeadLettersStayBounded(t *testing.T) {
	e, err := NewEngine(NewEngineConfig())
	assert.NoError(t, err)

	subscribed := make(chan struct{})
	pings := make(chan int)
	letters := make(chan DeadLetterEvent)
	marks := make(chan bool)
	timeout := time.After(3 * time.Second)

	e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case DeadLetterEvent:
			letters <- msg
		case boundPing:
			if msg.n > 0 {
				pings <- msg.n
			}
		case boundMark:
			marks <- msg.final
		}
	}, "monitor")

	sub := e.SpawnFunc(func(c *Context) {
		switch msg := c.Message().(type) {
		case Started:
			c.Engine().Subscribe(c.PID())
		case boundPing:
			if msg.n == 0 {
				close(subscribed)
			}
		}
	}, "sub", WithID("gone"))

	// Spawn queued both eventSub messages before this probe, so one
	// delivery means the subscription is active.
	e.BroadcastEvent(boundPing{})
	select {
	case <-subscribed:
	case <-timeout:
		t.Fatal("timed out waiting for subscription")
	}

	<-e.Poison(sub).Done()
	for n := 1; n <= 3; n++ {
		e.BroadcastEvent(boundPing{n: n})
	}

	delivered, dead, nested := 0, 0, 0
	flushed := false
	for {
		select {
		case <-pings:
			delivered++
		case dl := <-letters:
			// Forwarding a mark to the dead PID creates one ordinary dead
			// letter. It is not one of the four under test.
			if _, ok := dl.Message.(boundMark); ok {
				break
			}
			dead++
			if _, ok := dl.Message.(DeadLetterEvent); ok {
				nested++
			}
			if nested != 0 || dead > 4 {
				t.Fatalf("dead letters = %d, nested = %d, want 4 and 0", dead, nested)
			}
		case final := <-marks:
			if !final {
				// The flush cannot run until the receive that delivered the
				// last counted letter returns. A nested letter queued by
				// that receive is already ahead of the sentinel.
				e.BroadcastEvent(boundMark{final: true})
				break
			}
			assert.Equal(t, 3, delivered)
			assert.Equal(t, 4, dead)
			assert.Equal(t, 0, nested)
			return
		case <-timeout:
			t.Fatalf("timed out delivered=%d dead=%d nested=%d", delivered, dead, nested)
		}
		if !flushed && delivered >= 3 && dead >= 4 {
			flushed = true
			e.BroadcastEvent(boundMark{})
		}
	}
}
