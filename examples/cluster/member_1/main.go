package main

import (
	"fmt"
	"log"

	"github.com/fancom/hollywood/actor"
	"github.com/fancom/hollywood/cluster"
	"github.com/fancom/hollywood/examples/cluster/shared"
	"github.com/fancom/hollywood/remote"
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
	c.RegisterKind("playerSession", shared.NewPlayer, cluster.NewKindConfig())

	eventPID := c.Engine().SpawnFunc(func(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case cluster.ActivationEvent:
			fmt.Println("got activation event")
		case cluster.MemberJoinEvent:
			if msg.Member.ID == "B" {
				config := cluster.NewActivationConfig().
					WithID("bob").
					WithRegion("us-west")
				playerPID := c.Activate("playerSession", config)
				msg := &remote.TestMessage{Data: []byte("hello from member 1")}
				ctx.Send(playerPID, msg)
			}
		}
	}, "event")
	c.Engine().Subscribe(eventPID)

	c.Start()
	select {}
}
