package main

import (
	"fmt"
	"log"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/remote"
)

type Player struct{}

func NewPlayer() actor.Producer {
	return func() actor.Receiver {
		return &Player{}
	}
}

func (p *Player) Receive(c *actor.Context) {

}

func main() {
	remote := remote.New("127.0.0.1:3000", nil)
	e, err := actor.NewEngine(&actor.EngineConfig{Remote: remote})
	if err != nil {
		log.Fatal(err)
	}
	cfg := cluster.Config{
		ClusterProvider: cluster.NewSelfManagedProvider(),
		ID:              "A",
		Region:          "eu-west",
		Engine:          e,
	}
	c, err := cluster.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	c.RegisterKind("player", NewPlayer(), nil)
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
	pid := c.Activate("player", &cluster.ActivationConfig{ID: "1"})
	fmt.Println(pid)
	select {}
}
