package cluster

import (
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
)

const (
	ClusterAgentID = "agent"
	ProviderID     = "provider"
)

// Producer is a function that can produce an actor.Producer.
// Pretty simple, but yet powerfull tool to construct receivers
// depending on Cluster.
type Producer func(c *Cluster) actor.Producer

type Provider interface {
	Start(*Cluster) error
	Stop() error
}

type Config struct {
	ClusterName     string
	ID              string
	Engine          *actor.Engine
	ClusterProvider Producer
}

type Cluster struct {
	name string
	id   string

	provider    Producer
	engine      *actor.Engine
	agentPID    *actor.PID
	providerPID *actor.PID

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
		provider: cfg.ClusterProvider,
		engine:   cfg.Engine,
		kinds:    make(map[string]*Kind),
	}, nil
}

func (c *Cluster) Start() error {
	c.agentPID = c.engine.Spawn(NewAgent(c), "cluster/"+c.id)
	c.providerPID = c.engine.Spawn(c.provider(c), "cluster/"+c.id+"/provider")
	return nil
}

// Activate actives the given kind with the given id in the cluster.
func (c *Cluster) Activate(kind string, id string) *actor.PID {
	resp, err := c.engine.Request(c.agentPID, activate{kind: kind, id: id}, time.Millisecond*100).Result()
	if err != nil {
		slog.Error("activation failed", "err", err)
		return nil
	}
	return resp.(*actor.PID)
}

func (c *Cluster) RegisterKind(name string, producer actor.Producer, opts KindOpts) {
	c.kinds[name] = NewKind(name, producer, opts)
}

// PID returns the reachable actor process id, which is the Agent actor.
func (c *Cluster) PID() *actor.PID {
	return c.agentPID
}

// Member return the member info of this cluster.
func (c *Cluster) Member() *Member {
	return &Member{
		ID:    c.id,
		Host:  c.engine.Address(),
		Kinds: c.kindsToSlice(),
	}
}

func (c *Cluster) HasKind(kind string) bool {
	_, ok := c.kinds[kind]
	return ok
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
