package cluster

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"
)

// Producer is a function that can produce an actor.Producer.
// Pretty simple, but yet powerfull tool to construct receivers
// depending on Cluster.
type Producer func(c *Cluster) actor.Producer

// Config holds the cluster configuration
type Config struct {
	// The individual ID of this specific node
	ID string
	// The region this node is hosted
	Region string

	ActivationStrategy ActivationStrategy
	Engine             *actor.Engine
	ClusterProvider    Producer
}

type Cluster struct {
	id     string
	region string

	provider    Producer
	engine      *actor.Engine
	agentPID    *actor.PID
	providerPID *actor.PID

	isStarted bool

	activationStrategy ActivationStrategy

	kinds []kind
}

func New(cfg Config) (*Cluster, error) {
	if cfg.ActivationStrategy == nil {
		cfg.ActivationStrategy = DefaultActivationStrategy{}
	}
	if len(cfg.ID) == 0 {
		cfg.ID = uuid.New().String()
	}
	if len(cfg.Region) == 0 {
		return nil, fmt.Errorf("cannot start cluster without a region")
	}
	return &Cluster{
		id:                 cfg.ID,
		region:             cfg.Region,
		provider:           cfg.ClusterProvider,
		engine:             cfg.Engine,
		kinds:              []kind{},
		activationStrategy: cfg.ActivationStrategy,
	}, nil
}

// Start the cluster
func (c *Cluster) Start() error {
	c.agentPID = c.engine.Spawn(NewAgent(c), "cluster/"+c.id)
	c.providerPID = c.engine.Spawn(c.provider(c), "cluster/"+c.id+"/provider")
	c.isStarted = true
	return nil
}

// TODO: Spawn an actor with cluster awareness.
func (c *Cluster) Spawn(p actor.Producer, kind string, id string) *actor.PID {
	return nil
}

type ActivationConfig struct {
	// if empty, a unique identifier will be generated.
	ID     string
	Region string
}

// Activate actives the given actor kind with an optional id. If there is no id
// given, the engine will create an unique id automatically.
func (c *Cluster) Activate(kind string, config ActivationConfig) *actor.PID {
	resp, err := c.engine.Request(c.agentPID, activate{kind: kind, id: config.ID}, time.Millisecond*100).Result()
	if err != nil {
		slog.Error("activation failed", "err", err)
		return nil
	}
	pid, ok := resp.(*actor.PID)
	if !ok {
		slog.Warn("activation expected response of *actor.PID", "got", reflect.TypeOf(resp))
		return nil
	}
	return pid
}

// Deactivate deactivates the given PID.
func (c *Cluster) Deactivate(pid *actor.PID) {
	c.engine.Send(c.agentPID, deactivate{pid: pid})
}

// RegisterKind registers a new actor/receiver kind that can be spawned from any node
// on the cluster.
// NOTE: Kinds can only be registered if the cluster is not running.
func (c *Cluster) RegisterKind(name string, producer actor.Producer, config KindConfig) {
	if c.isStarted {
		slog.Warn("trying to register new kind on a running cluster")
		return
	}
	kind := newKind(name, producer, config)
	c.kinds = append(c.kinds, kind)
}

// HasLocalKind returns true if this members of the cluster has the kind
// locally registered.
func (c *Cluster) HasKindLocal(name string) bool {
	for _, kind := range c.kinds {
		if kind.name == name {
			return true
		}
	}
	return false
}

// PID returns the reachable actor process id, which is the Agent actor.
func (c *Cluster) PID() *actor.PID {
	return c.agentPID
}

// Member returns the member info of this node.
func (c *Cluster) Member() *Member {
	kinds := make([]string, len(c.kinds))
	for i := 0; i < len(c.kinds); i++ {
		kinds[i] = c.kinds[i].name
	}
	m := &Member{
		ID:    c.id,
		Host:  c.engine.Address(),
		Kinds: kinds,
	}
	return m
}

// Engine returns the actor engine.
func (c *Cluster) Engine() *actor.Engine {
	return c.engine
}
