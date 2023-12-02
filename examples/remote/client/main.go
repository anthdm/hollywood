package main

import (
	"fmt"
	"github.com/anthdm/hollywood/log"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

func main() {
	e := actor.NewEngine(actor.EngineOptLogger(log.Debug()))
	r := remote.New(e, remote.Config{ListenAddr: "127.0.0.1:3000"})
	err := e.WithRemote(r)
	defer r.Stop()
	if err != nil {
		panic(err)
	}

	pid := actor.NewPID("127.0.0.1:4000", "server")
	for {
		e.Send(pid, &msg.Message{Data: "hello!"})
		fmt.Println("sent message")
		time.Sleep(time.Second)
	}
}
