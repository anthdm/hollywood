package discovery

import (
	"github.com/grandcat/zeroconf"
)

const (
	serviceName = "_actor.hollywood_"
	domain      = "local."
	host        = "pc1"
)

type announcer struct {
	id     string
	ip     []string
	port   int
	server *zeroconf.Server
}

func newAnnouncer(cfg *options) *announcer {
	ret := &announcer{
		id:   cfg.id,
		ip:   cfg.ip,
		port: cfg.port,
	}
	return ret
}

func (a *announcer) start() {
	server, err := zeroconf.RegisterProxy(a.id, serviceName, domain, a.port, host, a.ip, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}
	a.server = server
}

func (a *announcer) shutdown() {
	a.server.Shutdown()
}
