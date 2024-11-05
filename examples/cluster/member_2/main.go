package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/fancom/hollywood/actor"
	"github.com/fancom/hollywood/cluster"
	"github.com/fancom/hollywood/examples/cluster/shared"
)

// Member 2 of the cluster
func main() {
	config := cluster.NewConfig().
		WithID("B").
		WithListenAddr("127.0.0.1:3001").
		WithRegion("us-west")
	c, err := cluster.New(config)
	if err != nil {
		log.Fatal(err)
	}
	eventPID := c.Engine().SpawnFunc(func(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case *cluster.Activation:
			fmt.Println("got activation event", msg)
		default:
			fmt.Println("got", reflect.TypeOf(msg))
		}
	}, "event")

	c.Engine().Subscribe(eventPID)
	c.RegisterKind("playerSession", shared.NewPlayer, cluster.NewKindConfig())
	c.Start()
	select {}
}
