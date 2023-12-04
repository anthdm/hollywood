package cluster

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type Agent struct {
}

func NewAgent() actor.Receiver {
	return &Agent{}
}

func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("cluster agent started")
		_ = msg
	}
}
