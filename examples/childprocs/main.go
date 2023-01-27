package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
)

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
		fmt.Println("bar started with initial state:", r.data)
		_ = msg
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
		_ = msg
	case message:
		fmt.Println("received and starting bar:", msg.data)
		// r.barPID = ctx.Engine().Spawn(newBarReceiver(msg.data), "bar", msg.data)
		r.barPID = ctx.SpawnChild(newBarReceiver(msg.data), "bar", msg.data)

		fmt.Println(r.barPID)
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
	// for i := 0; i < 10; i++ {
	e.Send(pid, message{data: fmt.Sprintf("msg_%d", 1)})
	time.Sleep(time.Millisecond)
	e.Poison(pid)
	// }
	time.Sleep(time.Millisecond * 100000)
}
