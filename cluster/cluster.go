package cluster

import (
	"log/slog"
	"os"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"github.com/anthdm/hollywood/remote"
)

type Producer func(c *Cluster) actor.Producer

type Config struct {
	ID               string
	ListenAddr       string
	ProviderProducer Producer
	Engine           *actor.Engine
}

type Cluster struct {
	ID string

	engine *actor.Engine
	remote *remote.Remote

	providerProducer Producer
	providerPID      *actor.PID
	agentPID         *actor.PID
}

func New(cfg Config) *Cluster {
	remote := remote.New(remote.Config{
		ListenAddr: cfg.ListenAddr,
	})

	if len(cfg.ID) == 0 {
		cfg.ID = "TODO SOMETHING RANDOM"
	}
	if cfg.ProviderProducer == nil {
		cfg.ProviderProducer = NewSelfManagedProvider
	}

	return &Cluster{
		ID:               cfg.ID,
		remote:           remote,
		providerProducer: cfg.ProviderProducer,
		engine:           cfg.Engine,
	}
}

func (c *Cluster) Start() error {
	lh := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelDebug)
	l := log.NewLogger("s", lh)
	c.engine = c.engine

	c.providerPID = c.engine.Spawn(c.providerProducer(c), c.ID, actor.WithTags("provider"))
	c.agentPID = c.engine.Spawn(NewAgent, c.ID, actor.WithTags("agent"))

	return c.remote.Start(c.engine, l)
}

type Member struct {
	ID   string
	Host string
	Port int
}

func (c *Cluster) MemberJoin(member Member) {
	c.engine.Send(c.agentPID, &MemberJoin{})
}
