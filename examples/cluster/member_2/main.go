package main

import (
	"log"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/examples/cluster/shared"
	"github.com/anthdm/hollywood/remote"
)

// Member 2 of the cluster
func main() {
	bootstrapAddr := cluster.MemberAddr{
		ListenAddr: "127.0.0.1:3000",
		ID:         "A",
	}
	r := remote.New("127.0.0.1:3001", nil)
	e, err := actor.NewEngine(&actor.EngineOpts{Remote: r})
	if err != nil {
		log.Fatal(err)
	}
	cluster, err := cluster.New(cluster.Config{
		ID:                 "B",
		Engine:             e,
		Region:             "us-west",
		ClusterProvider:    cluster.NewSelfManagedProvider(bootstrapAddr),
		ActivationStrategy: shared.RegionBasedActivationStrategy("eu-west"),
	})
	if err != nil {
		log.Fatal(err)
	}
	cluster.RegisterKind("playerSession", shared.NewPlayer, nil)
	cluster.Start()
	select {}
}
