package cluster

import (
	fmt "fmt"
	"log/slog"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

// pick a reasonable timeout so nodes of long distance networks (should) work.
var defaultRequestTimeout = time.Second

// Producer is a function that produces an actor.Producer given a *cluster.Cluster.
// Pretty simple, but yet powerfull tool to construct receivers that are depending on Cluster.
type Producer func(c *Cluster) actor.Producer

// Config holds the cluster configuration
type Config struct {
	listenAddr         string
	id                 string
	region             string
	activationStrategy ActivationStrategy
	engine             *actor.Engine
	provider           Producer
	requestTimeout     time.Duration
}

// NewConfig returns a Config that is initialized with default values.
func NewConfig() Config {
	return Config{
		listenAddr:         getRandomListenAddr(),
		id:                 fmt.Sprintf("%d", rand.Intn(math.MaxInt)),
		region:             "default",
		activationStrategy: NewDefaultActivationStrategy(),
		provider:           NewSelfManagedProvider(NewSelfManagedConfig()),
		requestTimeout:     defaultRequestTimeout,
	}
}

// WithRequestTimeout set's the maximum duration of how long a request
// can take between members of the cluster.
//
// Defaults to 1 second to support communication between nodes in
// other regions.
func (config Config) WithRequestTimeout(d time.Duration) Config {
	config.requestTimeout = d
	return config
}

// WithProvider set's the cluster provider.
//
// Defaults to the SelfManagedProvider.
func (config Config) WithProvider(p Producer) Config {
	config.provider = p
	return config
}

// WithEngine set's the internal actor engine that will be used
// to power the actors running on the node.
//
// If no engine is given the cluster will instanciate a new
// engine and remote.
func (config Config) WithEngine(e *actor.Engine) Config {
	config.engine = e
	return config
}

// TODO: Still not convinced about the name "ActivationStrategy".
// TODO: Document this more.
// WithActivationStrategy
func (config Config) WithActivationStrategy(s ActivationStrategy) Config {
	config.activationStrategy = s
	return config
}

// WithListenAddr set's the listen address of the underlying remote.
//
// Defaults to a random port number.
func (config Config) WithListenAddr(addr string) Config {
	config.listenAddr = addr
	return config
}

// WithID set's the ID of this node.
//
// Defaults to a random generated ID.
func (config Config) WithID(id string) Config {
	config.id = id
	return config
}

// WithRegion set's the region where the member will be hosted.
//
// Defaults to "default"
func (config Config) WithRegion(region string) Config {
	config.region = region
	return config
}

// Cluster...
type Cluster struct {
	config      Config
	engine      *actor.Engine
	agentPID    *actor.PID
	providerPID *actor.PID
	isStarted   bool
	kinds       []kind
}

func New(config Config) (*Cluster, error) {
	if config.engine == nil {
		remote := remote.New(config.listenAddr, nil)
		e, err := actor.NewEngine(&actor.EngineConfig{Remote: remote})
		if err != nil {
			return nil, err
		}
		config.engine = e
	}
	c := &Cluster{
		config: config,
		engine: config.engine,
		kinds:  make([]kind, 0),
	}
	return c, nil
}

// Start the cluster
func (c *Cluster) Start() {
	c.agentPID = c.engine.Spawn(NewAgent(c), "cluster", actor.WithID(c.config.id))
	c.providerPID = c.engine.Spawn(c.config.provider(c), "provider", actor.WithID(c.config.id))
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
	resp, err := c.engine.Request(c.agentPID, msg, c.config.requestTimeout).Result()
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
	resp, err := c.engine.Request(c.agentPID, getMembers{}, c.config.requestTimeout).Result()
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
	resp, err := c.engine.Request(c.agentPID, getKinds{}, c.config.requestTimeout).Result()
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
	resp, err := c.engine.Request(c.agentPID, getActive{id: id}, c.config.requestTimeout).Result()
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
		ID:     c.config.id,
		Host:   c.engine.Address(),
		Kinds:  kinds,
		Region: c.config.region,
	}
	return m
}

// Engine returns the actor engine.
func (c *Cluster) Engine() *actor.Engine {
	return c.engine
}

// Region return the region of the cluster.
func (c *Cluster) Region() string {
	return c.config.region
}

// ID returns the ID of the cluster.
func (c *Cluster) ID() string {
	return c.config.id
}

// Address returns the host/address of the cluster.
func (c *Cluster) Address() string {
	return c.agentPID.Address
}

// PID returns the reachable actor process id, which is the Agent actor.
func (c *Cluster) PID() *actor.PID {
	return c.agentPID
}

func getRandomListenAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", rand.Intn(50000)+10000)
}
