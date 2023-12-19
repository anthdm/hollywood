package main

import (
	"flag"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/mdns/chat"
	"github.com/anthdm/hollywood/examples/mdns/discovery"
	"github.com/anthdm/hollywood/remote"
)

var (
	port = flag.Int("port", 4001, "Set the port the service is listening to.")
	ip   = flag.String("ip", "127.0.0.1", "Set IP a service should be reachable.")
)

func main() {
	flag.Parse()

	rem := remote.New(fmt.Sprintf("%s:%d", *ip, *port), nil)
	engine, err := actor.NewEngine(&actor.EngineOpts{Remote: rem})
	if err != nil {
		panic(err)
	}

	engine.Spawn(chat.New(), "chat")

	// starts mdns discovery
	engine.Spawn(discovery.NewMdnsDiscovery(
		discovery.WithAnnounceAddr(*ip, *port),
	), "mdns")

	// Clean exit.
	select {}
}
