package discovery

import (
	"fmt"
	"time"
)

type DiscoveryOption func(*discoveryOptions)

type discoveryOptions struct {
	id   string   // engine global ID
	ip   []string // engine's ip to listen on
	port int      // engine's port to accept conns
}

func applyDiscoveryOptions(opts ...DiscoveryOption) *discoveryOptions {
	ret := &discoveryOptions{
		id:   fmt.Sprintf("engine_%d", time.Now().UnixNano()),
		ip:   make([]string, 0),
		port: 0,
	}

	for _, opt := range opts {
		opt(ret)
	}

	return ret
}

// Engine's IP and Port information to announce
func WithAnnounceAddr(ip string, p int) DiscoveryOption {
	return func(ao *discoveryOptions) {
		ao.ip = append(ao.ip, ip)
		ao.port = p
	}
}
