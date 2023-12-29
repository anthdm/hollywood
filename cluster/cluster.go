package cluster

import (
	fmt "fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"
)

// requestTimeout defines the default timeout duration for requests within the cluster.
var requestTimeout = time.Millisecond * 50

// Producer is a function that can produce an actor.Producer.
// Pretty simple, but yet powerfull tool to construct receivers
// depending on Cluster.
type Producer func(c *Cluster) actor.Producer

// Config holds the configuration settings for a cluster.
// It defines parameters that are essential for initializing and managing a cluster node.
type Config struct {
	ID string // The unique identifier for this specific cluster node.

	Region string // The geographic or logical region where this node is hosted.

	ActivationStrategy ActivationStrategy // The strategy used for activating entities within the cluster.

	Engine *actor.Engine // The actor engine associated with this cluster node, responsible for managing actors.

	ClusterProvider Producer // A Producer function that creates an actor.Producer, used to construct actor receivers for the cluster.
}

// Cluster represents a cluster configuration and state in the actor system.
// It contains all the necessary information and components to manage a cluster.
type Cluster struct {
	id     string // id is the unique identifier for the cluster.
	region string // region specifies the geographic or logical region of the cluster.

	provider    Producer      // provider is a function that creates an actor.Producer, used for constructing receivers.
	engine      *actor.Engine // engine is the actor engine associated with this cluster, managing the lifecycle of actors.
	agentPID    *actor.PID    // agentPID is the process identifier of the cluster agent, which manages cluster operations.
	providerPID *actor.PID    // providerPID is the process identifier for the provider actor.

	isStarted bool // isStarted indicates whether the cluster has been started.

	activationStrategy ActivationStrategy // activationStrategy defines the strategy for activating actors within the cluster.

	kinds []kind // kinds is a slice of kinds (types of actors) available in the cluster.
}

// New creates and initializes a new Cluster based on the provided Config.
// It returns a pointer to the Cluster or an error if any configuration issue occurs.
func New(cfg Config) (*Cluster, error) {
	// Validate that the engine parameter is provided.
	if cfg.Engine == nil {
		return nil, fmt.Errorf("engine parameter not provided")
	}

	// Validate that the cluster provider parameter is provided.
	if cfg.ClusterProvider == nil {
		return nil, fmt.Errorf("cluster provider parameter not provided")
	}

	// If no activation strategy is provided, use the default activation strategy.
	if cfg.ActivationStrategy == nil {
		cfg.ActivationStrategy = DefaultActivationStrategy()
	}

	// Generate a unique ID if not provided in the configuration.
	if len(cfg.ID) == 0 {
		cfg.ID = uuid.New().String()
	}

	// Set a default region if not provided in the configuration.
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
	return
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
// It returns the PID of the activated actor or nil if activation fails.
func (c *Cluster) Activate(kind string, config *ActivationConfig) *actor.PID {
	// Use a default configuration if none is provided.
	if config == nil {
		config = &ActivationConfig{}
	}

	// Create an activate message with the specified kind, id, and region.
	msg := activate{
		kind:   kind,          // The kind of actor to activate.
		id:     config.ID,     // The ID of the actor, or empty if an automatic ID should be generated.
		region: config.Region, // The region where the actor should be activated.
	}

	// Send a request to the agent PID to activate the actor, and get the result.
	resp, err := c.engine.Request(c.agentPID, msg, requestTimeout).Result()

	// Log an error and return nil if the activation request fails.
	if err != nil {
		slog.Error("activation failed", "err", err)
		return nil
	}

	// Check if the response is of type *actor.PID, indicating successful activation.
	pid, ok := resp.(*actor.PID)
	if !ok {
		// Warn if the response is not of the expected type and return nil.
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

// GetActivated requests the activation status of an actor by its ID within the cluster.
// It returns the actor's PID if the actor is activated, or nil if not activated or in case of an error.
func (c *Cluster) GetActivated(id string) *actor.PID {

	// Send a request to the agent PID to get the active status of an actor with the specified ID.
	resp, err := c.engine.Request(c.agentPID, getActive{id: id}, requestTimeout).Result()
	if err != nil {
		return nil
	}

	// If the response is of type *actor.PID, it means the actor is activated and its PID is returned.
	if res, ok := resp.(*actor.PID); ok {
		return res
	}

	// Return nil if the actor is not activated or the response is not of the expected type.
	return nil
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
