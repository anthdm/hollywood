package main

import (
	"time"

	"github.com/stevohuncho/hollywood/actor"
	"github.com/stevohuncho/hollywood/examples/remote/msg"
	"github.com/stevohuncho/hollywood/remote"
)

func main() {
	e := actor.NewEngine()
	r := remote.New(e, remote.Config{ListenAddr: "127.0.0.1:3000"})
	e.WithRemote(r)

	pid := actor.NewPID("127.0.0.1:4000", "server")
	for {
		e.Send(pid, &msg.Message{Data: "hello!"})
		time.Sleep(time.Second)
	}
}
