package cluster

import (
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
)

func makeCluster(t *testing.T, addr string, id string, bootstrapNodes ...string) *Cluster {
	e, _ := actor.NewEngine()
	cfg := Config{
		ListenAddr:       addr,
		ProviderProducer: NewSelfManagedProvider(bootstrapNodes...),
		ID:               id,
		Engine:           e,
	}
	return New(cfg)
}

func TestClusterDev(t *testing.T) {
	c1 := makeCluster(t, "localhost:3001", "A")
	c2 := makeCluster(t, "localhost:3002", "B")
	c1.Start()
	c2.Start()
	time.Sleep(time.Second)
}
