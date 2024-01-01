package main

import (
	"log"

	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/examples/cluster/shared"
)

// Member 2 of the cluster
func main() {
	config := cluster.NewConfig().
		WithID("B").
		WithListenAddr("127.0.0.1:3001").
		WithRegion("us-west")
	cluster, err := cluster.New(config)
	if err != nil {
		log.Fatal(err)
	}
	cluster.RegisterKind("playerSession", shared.NewPlayer, nil)
	cluster.Start()
	select {}
}
