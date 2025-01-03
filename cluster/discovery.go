package cluster

import (
	"context"
	"net"
	"runtime"

	"github.com/grandcat/zeroconf"
	"github.com/oleksandr/bonjour"
)

type ServiceRecord struct {
	Instance string `json:"name"`
	Service  string `json:"type"`
	Domain   string `json:"domain"`
}

type ServiceEntry struct {
	ServiceRecord
	HostName string   `json:"hostname"`
	Port     int      `json:"port"`
	Text     []string `json:"text"`
	TTL      uint32   `json:"ttl"`
	AddrIPv4 []net.IP `json:"-"`
	AddrIPv6 []net.IP `json:"-"`
}

type Resolver interface {
	Browse(ctx context.Context, service string, domain string, entries chan<- *ServiceEntry) error
	Lookup(ctx context.Context, instance string, service string, domain string, entries chan<- *ServiceEntry) error
}

const (
	darwin = "darwin"
)

func NewResolver(ifs ...net.Interface) (Resolver, error) {
	switch runtime.GOOS {
	case darwin:
		var iface *net.Interface
		if len(ifs) > 0 {
			iface = &ifs[0]
		}
		r, err := bonjour.NewResolver(iface)
		if err != nil {
			return nil, err
		}

		return &BonjourResolver{
			Resolver: r,
		}, nil
	default:
		var opt []zeroconf.ClientOption
		if len(ifs) > 0 {
			opt = append(opt, zeroconf.SelectIfaces(ifs))
		}
		r, err := zeroconf.NewResolver(opt...)
		if err != nil {
			return nil, err
		}

		return &ZeroconfResolver{
			Resolver: r,
		}, nil
	}
}

type ZeroconfResolver struct {
	*zeroconf.Resolver
}

func (r *ZeroconfResolver) Browse(ctx context.Context, service string, domain string, entries chan<- *ServiceEntry) error {
	ch := make(chan *zeroconf.ServiceEntry, cap(entries))
	go func(zeroconfEntries <-chan *zeroconf.ServiceEntry) {
		defer close(entries)
		for e := range zeroconfEntries {
			entries <- &ServiceEntry{
				ServiceRecord: ServiceRecord{
					Instance: e.Instance,
					Service:  e.Service,
					Domain:   e.Domain,
				},
				HostName: e.HostName,
				Port:     e.Port,
				Text:     e.Text,
				TTL:      e.TTL,
				AddrIPv4: e.AddrIPv4,
				AddrIPv6: e.AddrIPv6,
			}
		}
	}(ch)

	return r.Resolver.Browse(ctx, service, domain, ch)
}

func (r *ZeroconfResolver) Lookup(ctx context.Context, instance string, service string, domain string, entries chan<- *ServiceEntry) error {
	ch := make(chan *zeroconf.ServiceEntry, cap(entries))
	go func(zeroconfEntries <-chan *zeroconf.ServiceEntry) {
		defer close(entries)
		for e := range zeroconfEntries {
			entries <- &ServiceEntry{
				ServiceRecord: ServiceRecord{
					Instance: e.Instance,
					Service:  e.Service,
					Domain:   e.Domain,
				},
				HostName: e.HostName,
				Port:     e.Port,
				Text:     e.Text,
				TTL:      e.TTL,
				AddrIPv4: e.AddrIPv4,
				AddrIPv6: e.AddrIPv6,
			}
		}
	}(ch)

	return r.Resolver.Lookup(ctx, instance, service, domain, ch)
}

type BonjourResolver struct {
	*bonjour.Resolver
}

func (r *BonjourResolver) Browse(_ context.Context, service string, domain string, entries chan<- *ServiceEntry) error {
	ch := make(chan *bonjour.ServiceEntry, cap(entries))
	go func(bonjourEntries chan *bonjour.ServiceEntry) {
		defer close(entries)
		for e := range bonjourEntries {
			entries <- &ServiceEntry{
				ServiceRecord: ServiceRecord{
					Instance: e.Instance,
					Service:  e.Service,
					Domain:   e.Domain,
				},
				HostName: e.HostName,
				Port:     e.Port,
				Text:     e.Text,
				TTL:      e.TTL,
				AddrIPv4: []net.IP{e.AddrIPv4},
				AddrIPv6: []net.IP{e.AddrIPv6},
			}
		}
	}(ch)

	return r.Resolver.Browse(service, domain, ch)
}

func (r *BonjourResolver) Lookup(_ context.Context, instance string, service string, domain string, entries chan<- *ServiceEntry) error {
	ch := make(chan *bonjour.ServiceEntry, cap(entries))
	go func(bonjourEntries chan *bonjour.ServiceEntry) {
		defer close(entries)
		for e := range bonjourEntries {
			entries <- &ServiceEntry{
				ServiceRecord: ServiceRecord{
					Instance: e.Instance,
					Service:  e.Service,
					Domain:   e.Domain,
				},
				HostName: e.HostName,
				Port:     e.Port,
				Text:     e.Text,
				TTL:      e.TTL,
				AddrIPv4: []net.IP{e.AddrIPv4},
				AddrIPv6: []net.IP{e.AddrIPv6},
			}
		}
	}(ch)

	return r.Resolver.Lookup(instance, service, domain, ch)
}

type Discovery interface {
	RegisterProxy(instance string, service string, domain string, port int, host string, ips []string, text []string, ifaces []net.Interface) (Shutdownable, error)
}

type ZeroconfDiscovery struct {
}

func NewDiscovery() Discovery {
	switch runtime.GOOS {
	case darwin:
		return new(BonjourDiscovery)
	default:
		return new(ZeroconfDiscovery)
	}
}

func (z *ZeroconfDiscovery) RegisterProxy(instance string, service string, domain string, port int, host string, ips []string, text []string, ifaces []net.Interface) (Shutdownable, error) {
	server, err := zeroconf.RegisterProxy(instance, service, domain, port, host, ips, text, ifaces)

	return server, err
}

type BonjourDiscovery struct {
}

func (z *BonjourDiscovery) RegisterProxy(instance string, service string, domain string, port int, host string, ips []string, text []string, ifaces []net.Interface) (Shutdownable, error) {
	var iface *net.Interface
	if len(ifaces) > 0 {
		iface = &ifaces[0]
	}
	server, err := bonjour.RegisterProxy(instance, service, domain, port, host, append(ips, "")[0], text, iface)

	return server, err
}
