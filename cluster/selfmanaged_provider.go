package cluster

import (
	"github.com/anthdm/hollywood/actor"
)

type SelfManagedProvider struct {
	cluster *Cluster
}

func NewSelfManagedProvider(c *Cluster) actor.Producer {
	return func() actor.Receiver {
		return &SelfManagedProvider{
			cluster: c,
		}
	}
}

func (p *SelfManagedProvider) Receive(c *actor.Context) {
	switch c.Message().(type) {

	}
}
