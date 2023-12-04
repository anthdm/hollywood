package main

import (
	"fmt"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

type server struct{}

func newServer() actor.Receiver {
	return &server{}
}

func (f *server) Receive(ctx *actor.Context) {
	switch m := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("server has started")
	case *actor.PID:
		fmt.Println("server has received:", m)
	case *msg.Message:
		fmt.Println("got message", m)
	}
}

func main() {
	r := remote.New(remote.Config{ListenAddr: "127.0.0.1:4000"})
	e, err := actor.NewEngine(actor.EngineOptRemote(r))
	if err != nil {
		panic(err)
	}

	e.Spawn(newServer, "server")
	select {}
}
