package main

import (
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

func main() {
	e := actor.NewEngine()
	r := remote.New(e, remote.Config{ListenAddr: "127.0.0.1:5000"})
	e.WithRemote(r)

	e.SpawnFunc(func(c *actor.Context) {
	}, "receiver", actor.WithInboxSize(1024*1024))

	time.Sleep(time.Minute * 10)
}
