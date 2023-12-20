package discovery

import (
	"fmt"
	"time"
)

type Option func(*options)

type options struct {
	id   string   // engine global ID
	ip   []string // engine's ip to listen on
	port int      // engine's port to accept conns
}

func applyDiscoveryOptions(opts ...Option) *options {
	ret := &options{
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
func WithAnnounceAddr(ip string, p int) Option {
	return func(ao *options) {
		ao.ip = append(ao.ip, ip)
		ao.port = p
	}
}
