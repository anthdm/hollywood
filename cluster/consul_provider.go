package cluster

import (
	fmt "fmt"
	"log"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

const (
	registerTTL = 4 * time.Second
	refreshTTL  = 2 * time.Second
	removeTTL   = 8 * time.Second
)

type ConsulProviderConfig struct {
	address string
}

func NewConsulProviderConfig() ConsulProviderConfig {
	return ConsulProviderConfig{
		address: "127.0.0.1:8500",
	}
}

func (c ConsulProviderConfig) WithAddress(address string) ConsulProviderConfig {
	c.address = address
	return c
}

type ConsulProvider struct {
	config    ConsulProviderConfig
	cluster   *Cluster
	client    *api.Client
	id        string
	prevIndex watch.BlockingParamVal
	quitch    chan struct{}
}

func (p *ConsulProvider) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		_ = msg
		if err := p.registerService(); err != nil {
			panic(err)
		}
		go p.watch()
		go p.updateTTL()
	case actor.Stopped:
		close(p.quitch)
	}
}

func NewConsulProvider(config ConsulProviderConfig) Producer {
	client, err := api.NewClient(&api.Config{
		Address: config.address,
	})
	if err != nil {
		log.Fatal(err)
	}
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &ConsulProvider{
				config:    config,
				prevIndex: watch.WaitIndexVal(0),
				client:    client,
				cluster:   c,
				quitch:    make(chan struct{}),
			}
		}
	}
}

func (p *ConsulProvider) registerService() error {
	var (
		config      = p.cluster.config
		meta        = map[string]string{}
		taggedAddrs = map[string]api.ServiceAddress{}
	)

	meta["name"] = config.id
	check := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: removeTTL.String(),
		TLSSkipVerify:                  true,
		TTL:                            registerTTL.String(),
		CheckID:                        p.memberID(),
	}
	p.id = check.CheckID

	host, portStr, _ := net.SplitHostPort(p.cluster.Address())
	port, _ := strconv.Atoi(portStr)

	reg := &api.AgentServiceRegistration{
		ID:                p.memberID(),
		Name:              "hollywood_actor",
		Tags:              p.cluster.kindsToString(),
		Address:           host,
		Port:              port,
		Meta:              meta,
		TaggedAddresses:   taggedAddrs,
		EnableTagOverride: true,
		Check:             check,
	}
	regopts := api.ServiceRegisterOpts{
		ReplaceExistingChecks: true,
	}

	return p.client.Agent().ServiceRegisterOpts(reg, regopts)
}

func (p *ConsulProvider) watch() {
	query := map[string]any{
		"type":        "service",
		"service":     "hollywood_actor",
		"passingonly": true,
	}

	plan, err := watch.Parse(query)
	if err != nil {
		slog.Warn("consul provider", "err", err.Error())
		return
	}
	plan.HybridHandler = p.onUpdate

	config := &api.Config{}
	if err := plan.RunWithConfig(config.Address, config); err != nil {
		panic(err)
	}
}

func (p *ConsulProvider) onUpdate(index watch.BlockingParamVal, msg any) {
	entries, ok := msg.([]*api.ServiceEntry)
	if !ok {
		return
	}

	var members []*Member
	for _, entry := range entries {
		if len(entry.Checks) > 0 && entry.Checks.AggregatedStatus() == api.HealthPassing {
			port := strconv.Itoa(entry.Service.Port)
			member := &Member{
				ID:    entry.Service.Meta["name"],
				Host:  entry.Service.Address + ":" + port,
				Kinds: entry.Service.Tags,
			}
			members = append(members, member)
		}
	}

	if !p.prevIndex.Equal(index) && len(members) > 0 {
		msg := Members{Members: members}
		p.cluster.engine.Send(p.cluster.PID(), &msg)
	}

	p.prevIndex = index
}

func (p *ConsulProvider) updateTTL() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			err := p.client.Agent().UpdateTTL(p.id, "", api.HealthPassing)
			if err != nil {
				slog.Warn("failed to update TTL", "err", err.Error())
			}
		case <-p.quitch:
			return
		}
	}
}

func (p *ConsulProvider) memberID() string {
	config := p.cluster.config
	host, port, _ := net.SplitHostPort(config.listenAddr)
	return fmt.Sprintf("%s.%s:%s", config.id, host, port)
}
