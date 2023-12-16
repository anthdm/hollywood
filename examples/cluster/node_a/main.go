package main

import (
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
	remote := remote.New(remote.Config{
		ListenAddr: ":3000",
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
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
	c.RegisterKind("player", NewPlayer(), cluster.KindOpts{})
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
	c.Activate("player", "1")
	select {}
}
