package cluster

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type SelfManagedProvider struct {
	cluster       *Cluster
	boostrapNodes []string
}

func NewSelfManagedProvider(bootstrapNodes ...string) Producer {
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &SelfManagedProvider{
				cluster:       c,
				boostrapNodes: bootstrapNodes,
			}
		}
	}
}

func (p *SelfManagedProvider) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("provider selfmanaged started")
	case *MemberJoin:
		slog.Info("new member joined the cluster", "id", msg.ID, "host", msg.Host, "port", msg.Port)
	}
}
