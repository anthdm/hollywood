package cluster

import (
	fmt "fmt"
	"log/slog"
	"math"
	"math/rand"
	"reflect"
	"slices"
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
	listenAddr     string
	id             string
	region         string
	engine         *actor.Engine
	provider       Producer
	requestTimeout time.Duration
}

// NewConfig returns a Config that is initialized with default values.
func NewConfig() Config {
	return Config{
		listenAddr:     getRandomListenAddr(),
		id:             fmt.Sprintf("%d", rand.Intn(math.MaxInt)),
		region:         "default",
		provider:       NewSelfManagedProvider(NewSelfManagedConfig()),
		requestTimeout: defaultRequestTimeout,
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

// Cluster allows you to write distributed actors. It combines Engine, Remote, and
// Provider which allows members of the cluster to send messages to eachother in a
// self discovering environment.
type Cluster struct {
	config      Config
	engine      *actor.Engine
	agentPID    *actor.PID
	providerPID *actor.PID
	isStarted   bool
	kinds       []kind
}

// New returns a new cluster given a Config.
func New(config Config) (*Cluster, error) {
	if config.engine == nil {
		remote := remote.New(config.listenAddr, remote.NewConfig())
		e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(remote))
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
func (c *Cluster) Stop() {
	<-c.engine.Poison(c.agentPID).Done()
	<-c.engine.Poison(c.providerPID).Done()
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

// Activate actives the registered kind in the cluster based on the given config.
// The actor does not need to be registered locally on the member if at least one
// member has that kind registered.
//
//	playerPID := cluster.Activate("player", cluster.NewActivationConfig())
func (c *Cluster) Activate(kind string, config ActivationConfig) *actor.PID {
	msg := activate{
		kind:   kind,
		config: config,
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

// RegisterKind registers a new actor that can be activated from any member
// in the cluster.
//
//	cluster.Register("player", NewPlayer, NewKindConfig())
//
// NOTE: Kinds can only be registered before the cluster is started.
func (c *Cluster) RegisterKind(kind string, producer actor.Producer, config KindConfig) {
	if c.isStarted {
		slog.Warn("failed to register kind", "reason", "cluster already started", "kind", kind)
		return
	}
	c.kinds = append(c.kinds, newKind(kind, producer, config))
}

// HasKindLocal returns true whether the node of the cluster has the kind locally registered.
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
		return slices.Contains(kinds, name)
	}
	return false
}

// GetActiveByKind returns all the actor PIDS that are active across the cluster
// by the given kind. If not kids can be found on the cluster it will return a slice
// with a nil pid.
//
// Why?
// This guarantees that the function will return a slice with at least 1 element.
// The nil pid can be blindly used to send any message that will end up in the deadLetter.
// High available and correct code to handle a full fault tolarent system would look something
// like the following:
//
// playerPids := cluster.GetActiveByKind("player")
//
//	for _, pid := range playerPids {
//	 engine.Send(pid, theMessage)
//	}
//
// engine.Send(playerPid)
//
// playerPids := c.GetActiveByKind("player")
// [127.0.0.1:34364/player/1 127.0.0.1:34365/player/2]
func (c *Cluster) GetActiveByKind(kind string) []*actor.PID {
	resp, err := c.engine.Request(c.agentPID, getActive{kind: kind}, c.config.requestTimeout).Result()
	if err != nil {
		return []*actor.PID{nil}
	}
	if res, ok := resp.([]*actor.PID); ok {
		if len(res) == 0 || res == nil {
			return []*actor.PID{nil}
		}
		return res
	}
	return []*actor.PID{nil}
}

// GetActiveByID returns the full PID by the given ID.
//
//	playerPid := c.GetActiveByID("player/1")
//	// 127.0.0.1:34364/player/1
func (c *Cluster) GetActiveByID(id string) *actor.PID {
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

func (c *Cluster) kindsToString() []string {
	items := make([]string, len(c.kinds))
	for i := range len(c.kinds) {
		items[i] = c.kinds[i].name
	}
	return items
}

func getRandomListenAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", rand.Intn(50000)+10000)
}
