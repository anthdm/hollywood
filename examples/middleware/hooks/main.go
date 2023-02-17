package main

import (
	"fmt"
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
	actor.PIDSeparator = "→"
	e := actor.NewEngine()
	pid := e.Spawn(newFoo, "foo", actor.WithMiddleware(WithHooks()))
	e.Send(pid, "Hello sailor!")
	time.Sleep(time.Second)
	e.Poison(pid)
	time.Sleep(time.Second)
}
