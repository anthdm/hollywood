package main

import (
	"fmt"
	"sync"

	"github.com/anthdm/hollywood/actor"
)

func main() {
	e := actor.NewEngine()
	pid := e.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case actor.Started:
			fmt.Println("started")
			_ = msg
		}
	}, "foo")
	wg := sync.WaitGroup{}
	wg.Add(1)

	eventSub := e.EventStream.Subscribe(func(event any) {
		switch evt := event.(type) {
		case *actor.DeadLetter:
			fmt.Printf("deadletter event to [%s] msg: %s\n", evt.Target, evt.Message)
		case *actor.Termination:
			fmt.Println("process terminated:", evt.PID)
			wg.Done()
		default:
			fmt.Println("received event", evt)
		}
	})

	deadPID := actor.NewPID("local", "bar")
	e.Send(deadPID, "hello")
	// Publish anything to the stream.
	e.EventStream.Publish([]byte("some dirty bytes"))
	e.Poison(pid)

	// Unsubscribe from the event stream
	defer e.EventStream.Unsubscribe(eventSub)
	wg.Wait()
}
