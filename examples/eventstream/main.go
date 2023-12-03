package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
)

// Custom event type that will be send over the event stream.
type MyCustomEvent struct {
	msg string
}

// Spawn 2 actors and subscribe them to the event stream.
// When we call engine.PublishEvent both actors will be notified.
func main() {
	e := actor.NewEngine()
	actorA := e.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case actor.Started:
			fmt.Println("actor A started")
		case MyCustomEvent:
			fmt.Printf("actorA: event => %+v\n", msg)
		}
	}, "actor_a")
	// Subscribe the actor to the event stream from outside of the actor itself.
	e.Subscribe(actorA)

	actorB := e.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case actor.Started:
			fmt.Println("actor B started")
			// Subscribe the actor to the event stream from inside the actor itself.
			c.Engine().Subscribe(c.PID())
		case MyCustomEvent:
			fmt.Printf("actorB: event => %+v\n", msg)
		}
	}, "actor_b")

	// Unsubscribing both actors from the event stream.
	defer func() {
		e.Unsubscribe(actorA)
		e.Unsubscribe(actorB)
	}()

	time.Sleep(time.Millisecond)
	e.BroadcastEvent(MyCustomEvent{msg: "Hello World!"})
	time.Sleep(time.Millisecond)
}
