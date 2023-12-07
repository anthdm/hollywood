package cluster

import (
	"github.com/anthdm/hollywood/actor"
)

const ClusterAgentID = "agent"

type Producer func(c *Cluster) actor.Producer

type Provider interface {
	Start(*Cluster) error
	Stop() error
}

type Config struct {
	ClusterName string
	ID          string
	Engine      *actor.Engine
	Provider    Provider
}

type Cluster struct {
	name string
	id   string

	provider Provider
	engine   *actor.Engine
	agentPID *actor.PID

	kinds map[string]*Kind
}

func New(cfg Config) (*Cluster, error) {
	if len(cfg.ClusterName) == 0 {
		cfg.ClusterName = "todo random"
	}
	if len(cfg.ID) == 0 {
		cfg.ID = "TODO SOMETHING RANDOM"
	}
	return &Cluster{
		id:       cfg.ID,
		name:     cfg.ClusterName,
		provider: cfg.Provider,
		engine:   cfg.Engine,
		kinds:    make(map[string]*Kind),
	}, nil
}

func (c *Cluster) Start() error {
	c.agentPID = c.engine.Spawn(NewAgent, "cluster/"+c.id)
	c.provider.Start(c)
	return nil
}

// PID returns the reachable actor process id.
func (c *Cluster) PID() *actor.PID {
	return nil
}

// Member return the member info of this cluster.
func (c *Cluster) Member() *Member {
	return &Member{
		ID:    c.id,
		Host:  c.engine.Address(),
		Kinds: c.kindsToSlice(),
	}
}

func (c *Cluster) kindsToSlice() []string {
	kinds := make([]string, len(c.kinds))
	i := 0
	for kind := range c.kinds {
		kinds[i] = kind
		i++
	}
	return kinds
}
