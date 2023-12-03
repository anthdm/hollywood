package main

import (
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

func main() {
	r := remote.New(remote.Config{ListenAddr: "127.0.0.1:3000"})
	e := actor.NewEngine(actor.EngineOptRemote(r))

	pid := actor.NewPID("127.0.0.1:4000", "server")
	for {
		e.Send(pid, &msg.Message{Data: "hello!"})
		time.Sleep(time.Second)
	}
}
