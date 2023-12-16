package main

import (
	"log"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/remote"
)

type Inventory struct{}

func NewInventory() actor.Producer {
	return func() actor.Receiver {
		return &Inventory{}
	}
}

func (i *Inventory) Receive(c *actor.Context) {

}

func main() {
	remote := remote.New(remote.Config{
		ListenAddr: ":3001",
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	memberAddr := cluster.MemberAddr{
		ListenAddr: "localhost:3000",
		ID:         "A",
	}
	cfg := cluster.Config{
		ClusterProvider: cluster.NewSelfManagedProvider(memberAddr),
		ID:              "B",
		Region:          "eu-east",
		Engine:          e,
	}
	c, err := cluster.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	c.RegisterKind("inventory", NewInventory(), cluster.KindOpts{})
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
	c.Activate("inventory", "1")
	select {}
}
