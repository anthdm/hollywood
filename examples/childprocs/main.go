package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
)

var restarts = 0

type barReceiver struct {
	data string
}

func newBarReceiver(data string) actor.Producer {
	return func() actor.Receiver {
		return &barReceiver{
			data: data,
		}
	}
}

func (r *barReceiver) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		time.Sleep(time.Second)
		if restarts < 2 {
			restarts++
			panic("I will need to restart")
		}
		fmt.Println("bar recovered and started with initial state:", r.data)
	case message:
		fmt.Println(msg.data)
	case actor.Stopped:
		fmt.Println("bar stopped")
	}
}

type fooReceiver struct {
	barPID *actor.PID
}

func newFooReceiver() actor.Receiver {
	return &fooReceiver{}
}

func (r *fooReceiver) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("foo started")
	case message:
		r.barPID = ctx.SpawnChild(newBarReceiver(msg.data), "bar", msg.data)
		fmt.Println("received and starting bar:", r.barPID)
	case actor.Stopped:
		fmt.Println("foo will stop")
	}
}

type message struct {
	data string
}

func main() {
	e := actor.NewEngine()
	pid := e.Spawn(newFooReceiver, "foo")
	e.Send(pid, message{data: fmt.Sprintf("msg_%d", 1)})
	time.Sleep(time.Second * 8)
	e.Poison(pid)
	time.Sleep(time.Second)
}
