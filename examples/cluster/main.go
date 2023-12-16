package main

import (
	"fmt"
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/remote"
)

func makeCluster(addr string, id string, members ...cluster.MemberAddr) *cluster.Cluster {
	remote := remote.New(remote.Config{
		ListenAddr: addr,
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	cfg := cluster.Config{
		ClusterProvider: cluster.NewSelfManagedProvider(members...),
		ID:              id,
		Region:          "eu-west",
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

func (p *Player) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case *actor.PID:
		fmt.Println("got", msg)
	}
}

func main() {
	c1 := makeCluster("localhost:3001", "A")
	c1.RegisterKind("inventory", NewInventory, cluster.KindOpts{})
	c1.RegisterKind("player", NewPlayer, cluster.KindOpts{})
	c1.Start()
	time.Sleep(time.Second)

	memberAddr := cluster.MemberAddr{
		ListenAddr: "localhost:3001",
		ID:         "A",
	}
	c2 := makeCluster("localhost:3002", "B", memberAddr)
	c2.RegisterKind("inventory", NewInventory, cluster.KindOpts{})
	c2.RegisterKind("player", NewPlayer, cluster.KindOpts{})
	c2.Start()

	time.Sleep(time.Second)

	cid := c2.Activate("player", "supermario")
	time.Sleep(time.Second)

	c3 := makeCluster("localhost:3003", "C", memberAddr)
	c3.Start()

	time.Sleep(time.Second * 2)
	c3.Spawn(NewInventory, "foobar", "inv")
	c3.Engine().Send(cid.PID, cid.PID)

	time.Sleep(time.Second)

	fmt.Println(c3.HasKind("player"))
	fmt.Println(c1.HasKind("player"))
	fmt.Println(c2.HasKind("player"))

	time.Sleep(time.Second * 10000)
}
