package cluster

import (
	"github.com/anthdm/hollywood/actor"
)

type Producer func(c *Cluster) actor.Producer

type Provider interface {
	Start(*Cluster) error
	Stop() error
}

type Config struct {
	ID       string
	Engine   *actor.Engine
	Provider Provider
}

type Cluster struct {
	id string

	engine   *actor.Engine
	provider Provider

	agentPID *actor.PID
}

func New(cfg Config) (*Cluster, error) {
	if len(cfg.ID) == 0 {
		cfg.ID = "TODO SOMETHING RANDOM"
	}
	return &Cluster{
		id:       cfg.ID,
		provider: cfg.Provider,
		engine:   cfg.Engine,
	}, nil
}

func (c *Cluster) Start() error {
	c.agentPID = c.engine.Spawn(NewAgent, "cluster", actor.WithTags(c.id, "agent"))
	c.provider.Start(c)
	return nil
}

func (c *Cluster) MemberJoin(member *Member) {
	c.engine.Send(c.agentPID, MemberJoin{
		Member: member,
	})
}

// Member return the member info of this cluster.
func (c *Cluster) Member() Member {
	return Member{
		ID:  c.id,
		PID: actor.NewPID(c.engine.Address(), "cluster", c.id, "agent"),
	}
}
