package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type message struct {
	data string
}

var wg sync.WaitGroup

type foo struct{}

func newFoo() actor.Receiver {
	return &foo{}
}

func (f *foo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("foo started")
	case *message:
		if msg.data == "failed" {
			panic("I failed processing this message")
		}
		wg.Done()
		fmt.Println("I restarted and processed the next one perfectly:", msg.data)
	}
}

func main() {
	engine, err := actor.NewEngine()
	if err != nil {
		panic(err)
	}
	pid := engine.Spawn(newFoo, "foo", actor.WithMaxRestarts(3))
	wg.Add(1)
	engine.Send(pid, &message{data: "failed"})
	time.Sleep(time.Millisecond)
	engine.Send(pid, &message{data: "hello world!"})
	wg.Wait()
}
