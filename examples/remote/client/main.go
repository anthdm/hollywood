package main

import (
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

func main() {
	r := remote.New(remote.Config{ListenAddr: "127.0.0.1:3000"})
	e, err := actor.NewEngine(actor.EngineOptRemote(r))
	if err != nil {
		panic(err)
	}

	// The server will be started with id "myserverid". Hence, let's create
	// the correct PID for it so its reachable.
	serverPID := actor.NewPID("127.0.0.1:4000", "server/myserverid")
	for {
		e.Send(serverPID, &msg.Message{Data: "hello!"})
		time.Sleep(time.Second)
	}
}
