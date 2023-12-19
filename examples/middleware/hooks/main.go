package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type Hooker interface {
	OnInit(*actor.Context)
	OnStart(*actor.Context)
	OnStop(*actor.Context)
	Receive(*actor.Context)
}

type foo struct{}

func newFoo() actor.Receiver {
	return &foo{}
}

func (f *foo) Receive(c *actor.Context) {}
func (f *foo) OnInit(c *actor.Context)  { fmt.Println("foo initialized") }
func (f *foo) OnStart(c *actor.Context) { fmt.Println("foo started") }
func (f *foo) OnStop(c *actor.Context)  { fmt.Println("foo stopped") }

func WithHooks() func(actor.ReceiveFunc) actor.ReceiveFunc {
	return func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(c *actor.Context) {
			switch c.Message().(type) {
			case actor.Initialized:
				c.Receiver().(Hooker).OnInit(c)
			case actor.Started:
				c.Receiver().(Hooker).OnStart(c)
			case actor.Stopped:
				c.Receiver().(Hooker).OnStop(c)
			}
			next(c)
		}
	}
}

func main() {
	// Create a new engine
	e, err := actor.NewEngine(nil)
	if err != nil {
		panic(err)
	}
	// Spawn the a new "foo" receiver with middleware.
	pid := e.Spawn(newFoo, "foo", actor.WithMiddleware(WithHooks()))
	// Send a message to foo
	e.Send(pid, "Hello sailor!")
	// We sleep here so we are sure foo received our message
	time.Sleep(time.Second)
	// Create a waitgroup so we can wait until foo has been stopped gracefully
	wg := &sync.WaitGroup{}
	e.Poison(pid, wg)
	wg.Wait()
}
