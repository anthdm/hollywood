package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fancom/hollywood/actor"
	"github.com/grandcat/zeroconf"
)

type mdns struct {
	id        string
	announcer *announcer
	resolver  *zeroconf.Resolver
	engine    *actor.Engine

	ctx      context.Context
	cancelFn context.CancelFunc
}

func NewMdnsDiscovery(opts ...Option) actor.Producer {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := applyDiscoveryOptions(opts...)
	announcer := newAnnouncer(cfg)
	return func() actor.Receiver {
		ret := &mdns{
			id:        cfg.id,
			announcer: announcer,
			ctx:       ctx,
			cancelFn:  cancel,
		}
		return ret
	}
}

func (d *mdns) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		d.engine = ctx.Engine()
		slog.Info("[DISCOVERY] initializing")
		d.ctx, d.cancelFn = context.WithCancel(context.Background())
		d.createResolver()
	case actor.Started:
		slog.Info("[DISCOVERY] starting discovery")
		go d.startDiscovery(d.ctx)
		d.announcer.start()
	case actor.Stopped:
		slog.Info("[DISCOVERY] stopping discovery")
		d.shutdown()
		d.cancelFn()
		_ = msg
	}
}

func (d *mdns) shutdown() {
	if d.announcer != nil {
		d.announcer.shutdown()
	}
	d.cancelFn()
}

func (d *mdns) createResolver() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(err)
	}
	d.resolver = resolver
}

// Starts multicast dns discovery process.
// Searches matching entries with `serviceName` and `domain`.
func (d *mdns) startDiscovery(ctx context.Context) {
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			d.sendDiscoveryEvent(entry)
		}
	}(entries)
	err := d.resolver.Browse(ctx, serviceName, domain, entries)
	if err != nil {
		slog.Error("[DISCOVERY] starting discovery failed", "err", err)
		panic(err)
	}
	<-ctx.Done()
}

// Sends discovered peer as `DiscoveryEvent` to event stream.
func (d *mdns) sendDiscoveryEvent(entry *zeroconf.ServiceEntry) {
	// avoid to discover myself
	if entry.Instance == d.id {
		return
	}
	event := &DiscoveryEvent{
		ID:   entry.Instance,
		Addr: []string{},
	}
	for _, addr := range entry.AddrIPv4 {
		event.Addr = append(event.Addr, fmt.Sprintf("%s:%d", addr.String(), entry.Port))
	}
	slog.Info("[DISCOVERY] remote discovered", "addrs", strings.Join(event.Addr, ","), "ID", entry.Instance)
	if d.engine == nil {
		slog.Error("[DISCOVERY] engine is nil")
		return
	}
	d.engine.BroadcastEvent(event)
}
