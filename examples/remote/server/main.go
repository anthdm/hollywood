package main

import (
	"fmt"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

type server struct{}

func newServer() actor.Receiver {
	return &server{}
}

func (f *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("server has started")
	case *actor.PID:
		fmt.Println("server has received:", msg)
	}
}

func main() {
	e := actor.NewEngine()
	r := remote.New(e, remote.Config{ListenAddr: "127.0.0.1:4000"})
	e.WithRemote(r)

	e.Spawn(newServer, "server")
	select {}
}
