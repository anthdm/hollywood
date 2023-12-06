package cluster

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type Agent struct {
	members map[*actor.PID]*Member
}

func NewAgent() actor.Receiver {
	return &Agent{}
}

func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("cluster agent started", "pid", c.PID())
	case *MemberJoin:
		slog.Info("new member joined the cluster", "id", msg.Member.ID, "pid", msg.Member.PID)
	}
}
