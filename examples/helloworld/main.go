package main

import (
	"fmt"
	"github.com/anthdm/hollywood/actor"
)

type message struct {
	data string
}

type foo struct{}

func newFoo() actor.Receiver {
	return &foo{}
}

func (f *foo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("actor started")
	case actor.Stopped:
		fmt.Println("actor stopped")
	case *message:
		fmt.Println("actor has received", msg.data)
	}
}

type info struct {
	data string
}

func main() {

	var ds []*info
	ds = append(ds, &info{data: "hello"})

	fmt.Printf("ds len: %d\n", len(ds))

	fmt.Printf("%+v\n", ds)
	engine, err := actor.NewEngine(nil)
	if err != nil {
		panic(err)
	}

	pid := engine.Spawn(newFoo, "my_actor")
	for i := 0; i < 3; i++ {
		engine.Send(pid, &message{data: "hello world!"})
	}
	engine.Poison(pid).Wait()
}
