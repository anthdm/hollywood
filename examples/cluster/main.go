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
		Provider: cluster.NewSelfManagedProvider(members...),
		ID:       id,
		Engine:   e,
	}
	c, err := cluster.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func main() {
	c1 := makeCluster("localhost:3001", "A")
	c1.Start()
	time.Sleep(time.Second)

	member := c1.Member()
	c2 := makeCluster("localhost:3002", "B", member)
	c2.Start()
	time.Sleep(time.Second * 500)
}
