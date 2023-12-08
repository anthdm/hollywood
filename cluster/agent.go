package cluster

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type Agent struct {
	members *Map[string, *Member]
	cluster *Cluster
}

func NewAgent(c *Cluster) actor.Producer {
	return func() actor.Receiver {
		return &Agent{
			members: NewMap[string, *Member](),
			cluster: c,
		}
	}
}

func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("cluster agent started", "pid", c.PID())
	case *Members:
		a.handleMembersJoin(msg.Members)
	}
}

func (a *Agent) handleMembersJoin(members []*Member) {
	// TODO: Do topology stuff right here.
	for _, member := range members {
		a.memberJoin(member)
	}
}

func (a *Agent) memberJoin(member *Member) {
	slog.Info("member joined", "we", a.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
	a.members.Add(member.ID, member)
}
