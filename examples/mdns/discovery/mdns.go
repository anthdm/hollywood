package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/stevohuncho/hollywood/actor"
	"github.com/stevohuncho/hollywood/log"
)

type mdns struct {
	id          string
	announcer   *announcer
	resolver    *zeroconf.Resolver
	eventStream *actor.EventStream

	ctx      context.Context
	cancelFn context.CancelFunc
}

func NewMdnsDiscovery(eventStream *actor.EventStream, opts ...DiscoveryOption) actor.Producer {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := applyDiscoveryOptions(opts...)
	announcer := newAnnouncer(cfg)
	return func() actor.Receiver {
		ret := &mdns{
			id:          cfg.id,
			announcer:   announcer,
			ctx:         ctx,
			cancelFn:    cancel,
			eventStream: eventStream,
		}
		return ret
	}
}

func (d *mdns) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		d.createResolver()
	case actor.Started:
		go d.startDiscovery(ctx)
		d.announcer.start()
	case actor.Stopped:
		d.shutdown()
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
func (d *mdns) startDiscovery(c *actor.Context) {
	ctx, cancel := context.WithCancel(d.ctx)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			d.sendDiscoveryEvent(entry)
		}
	}(entries)

	err := d.resolver.Browse(ctx, serviceName, domain, entries)
	if err != nil {
		log.Infow("[DISCOVERY] starting discovery failed", log.M{"err": err.Error()})
		panic(err)
	}
	<-ctx.Done()
}

// Sends discovered peer as `DiscoveryEvent` to event stream.
func (d *mdns) sendDiscoveryEvent(entry *zeroconf.ServiceEntry) {
	// avoid to discover myself
	if entry.Instance != d.id {
		event := &DiscoveryEvent{
			ID:   entry.Instance,
			Addr: []string{},
		}
		for _, addr := range entry.AddrIPv4 {
			event.Addr = append(event.Addr, fmt.Sprintf("%s:%d", addr.String(), entry.Port))
		}
		log.Infow("[DISCOVERY] remote discovered", log.M{"addrs": strings.Join(event.Addr, ","), "ID": entry.Instance})
		d.eventStream.Publish(event)
	}
}
