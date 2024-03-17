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

type foo struct{ done func() }

func (f *foo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("foo started")
	case *message:
		if msg.data == "failed" {
			panic("I failed processing this message")
		}
		fmt.Println("I restarted and processed the next one perfectly:", msg.data)
		f.done()
	}
}

func main() {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	pid := engine.Spawn(
		func() actor.Receiver { return &foo{done: wg.Done} },
		"foo", actor.WithMaxRestarts(3))
	engine.Send(pid, &message{data: "failed"})
	time.Sleep(time.Millisecond)
	engine.Send(pid, &message{data: "hello world!"})
	wg.Wait()
}
