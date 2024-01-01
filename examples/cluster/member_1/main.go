package main

import (
	"log"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/examples/cluster/shared"
	"github.com/anthdm/hollywood/remote"
)

// Member 1 of the cluster
func main() {
	config := cluster.NewConfig().
		WithID("A").
		WithListenAddr("127.0.0.1:3000").
		WithRegion("eu-west")
	c, err := cluster.New(config)
	if err != nil {
		log.Fatal(err)
	}
	c.RegisterKind("playerSession", shared.NewPlayer, nil)

	eventPID := c.Engine().SpawnFunc(func(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case cluster.MemberJoinEvent:
			if msg.Member.ID == "B" {
				msg := &cluster.ActivationConfig{
					ID:     "bob",
					Region: "us-west",
				}
				playerPID := c.Activate("playerSession", msg)
				ctx.Send(playerPID, &remote.TestMessage{Data: []byte("hello from member 1")})
			}
		}
	}, "event")
	c.Engine().Subscribe(eventPID)

	c.Start()
	select {}
}
