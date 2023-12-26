package cluster

import (
	fmt "fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"
)

var requestTimeout = time.Millisecond * 50

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
	if cfg.Engine == nil {
		return nil, fmt.Errorf("engine parameter not provided")
	}
	if cfg.ClusterProvider == nil {
		return nil, fmt.Errorf("cluster provider parameter not provided")
	}
	if cfg.ActivationStrategy == nil {
		cfg.ActivationStrategy = DefaultActivationStrategy()
	}
	if len(cfg.ID) == 0 {
		cfg.ID = uuid.New().String()
	}
	if len(cfg.Region) == 0 {
		cfg.Region = "default"
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
func (c *Cluster) Start() {
	c.agentPID = c.engine.Spawn(NewAgent(c), "cluster", actor.WithID(c.id))
	c.providerPID = c.engine.Spawn(c.provider(c), "provider", actor.WithID(c.id))
	c.isStarted = true
}

// Stop will shutdown the cluster poisoning all its actors.
func (c *Cluster) Stop() *sync.WaitGroup {
	wg := sync.WaitGroup{}
	c.engine.Poison(c.agentPID, &wg)
	c.engine.Poison(c.providerPID, &wg)
	return &wg
}

// Spawn an actor locally on the node with cluster awareness.
func (c *Cluster) Spawn(p actor.Producer, id string, opts ...actor.OptFunc) *actor.PID {
	pid := c.engine.Spawn(p, id, opts...)
	members := c.Members()
	for _, member := range members {
		c.engine.Send(member.PID(), &Activation{
			PID: pid,
		})
	}
	return pid
}

// TODO: Doc this when its more usefull.
type ActivationConfig struct {
	// if empty, a unique identifier will be generated.
	ID     string
	Region string
}

// Activate actives the given actor kind with an optional id. If there is no id
// given, the engine will create an unique id automatically.
func (c *Cluster) Activate(kind string, config *ActivationConfig) *actor.PID {
	if config == nil {
		config = &ActivationConfig{}
	}
	msg := activate{
		kind:   kind,
		id:     config.ID,
		region: config.Region,
	}
	resp, err := c.engine.Request(c.agentPID, msg, requestTimeout).Result()
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
func (c *Cluster) RegisterKind(name string, producer actor.Producer, config *KindConfig) {
	if c.isStarted {
		slog.Warn("trying to register new kind on a running cluster")
		return
	}
	if config == nil {
		config = &KindConfig{}
	}
	kind := newKind(name, producer, *config)
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

// Members returns all the members that are part of the cluster.
func (c *Cluster) Members() []*Member {
	resp, err := c.engine.Request(c.agentPID, getMembers{}, requestTimeout).Result()
	if err != nil {
		return []*Member{}
	}
	if res, ok := resp.([]*Member); ok {
		return res
	}
	return nil
}

// HasKind returns true whether the given kind is available for activation on
// the cluster.
func (c *Cluster) HasKind(name string) bool {
	resp, err := c.engine.Request(c.agentPID, getKinds{}, requestTimeout).Result()
	if err != nil {
		return false
	}
	if kinds, ok := resp.([]string); ok {
		for _, kind := range kinds {
			if kind == name {
				return true
			}
		}
	}
	return false
}

func (c *Cluster) GetActivated(id string) *actor.PID {
	resp, err := c.engine.Request(c.agentPID, getActive{id: id}, requestTimeout).Result()
	if err != nil {
		return nil
	}
	if res, ok := resp.(*actor.PID); ok {
		return res
	}
	return nil
}

// Member returns the member info of this node.
func (c *Cluster) Member() *Member {
	kinds := make([]string, len(c.kinds))
	for i := 0; i < len(c.kinds); i++ {
		kinds[i] = c.kinds[i].name
	}
	m := &Member{
		ID:     c.id,
		Host:   c.engine.Address(),
		Kinds:  kinds,
		Region: c.region,
	}
	return m
}

// Engine returns the actor engine.
func (c *Cluster) Engine() *actor.Engine {
	return c.engine
}

// Region return the region of the cluster.
func (c *Cluster) Region() string {
	return c.region
}

// ID returns the ID of the cluster.
func (c *Cluster) ID() string {
	return c.id
}

// Address returns the host/address of the cluster.
func (c *Cluster) Address() string {
	return c.agentPID.Address
}

// PID returns the reachable actor process id, which is the Agent actor.
func (c *Cluster) PID() *actor.PID {
	return c.agentPID
}
