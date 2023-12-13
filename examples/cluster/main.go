package main

import (
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/remote"
)

func makeCluster(addr string, id string, members ...*cluster.Member) *cluster.Cluster {
	remote := remote.New(remote.Config{
		ListenAddr: addr,
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	cfg := cluster.Config{
		ClusterName:     "My Cluster",
		ClusterProvider: cluster.NewSelfManagedProvider(members...),
		ID:              id,
		Engine:          e,
	}
	c, err := cluster.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

type Inventory struct{}

func NewInventory() actor.Receiver {
	return &Inventory{}
}

func (i *Inventory) Receive(c *actor.Context) {}

type Player struct{}

func NewPlayer() actor.Receiver {
	return &Player{}
}

func (p *Player) Receive(c *actor.Context) {}

func main() {
	c1 := makeCluster("localhost:3001", "A")
	c1.RegisterKind("inventory", NewInventory, cluster.KindOpts{})
	c1.RegisterKind("player", NewPlayer, cluster.KindOpts{})
	c1.Start()
	time.Sleep(time.Second)

	member := c1.Member()
	c2 := makeCluster("localhost:3002", "B", member)
	c2.RegisterKind("inventory", NewInventory, cluster.KindOpts{})
	c2.RegisterKind("player", NewPlayer, cluster.KindOpts{})
	c2.Start()

	time.Sleep(time.Second)

	c2.Activate("player", "supermario")
	time.Sleep(time.Second)

	c3 := makeCluster("localhost:3003", "C", member)
	c3.RegisterKind("inventory", NewInventory, cluster.KindOpts{})
	c3.Start()

	time.Sleep(time.Second * 10000)
}
