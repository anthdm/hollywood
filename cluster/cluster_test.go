package cluster

import (
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
)

func TestClusterDev(t *testing.T) {
	cfg := Config{
		ListenAddr:       "127.0.0.1:4000",
		ProviderProducer: NewSelfManagedProvider,
		ID:               "mycluster",
	}
	c := New(cfg)
	e, _ := actor.NewEngine()
	c.Start(e)
	time.Sleep(time.Second * 2)
}
