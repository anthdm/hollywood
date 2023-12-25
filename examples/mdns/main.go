package main

import (
	"flag"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/mdns/chat"
	"github.com/anthdm/hollywood/examples/mdns/discovery"
	"github.com/anthdm/hollywood/remote"
	"log/slog"
	"math/rand"
	"os"
)

var (
	port = flag.Int("port", 0, "Set the port the service is listening to.")
	ip   = flag.String("ip", "127.0.0.1", "Set IP a service should be reachable.")
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	flag.Parse()
	if *port == 0 {
		// pick a random port, 2000 and up.
		*port = rand.Intn(10000) + 2000
	}
	rem := remote.New(fmt.Sprintf("%s:%d", *ip, *port), nil)
	engine, err := actor.NewEngine(&actor.EngineConfig{Remote: rem})
	if err != nil {
		panic(err)
	}

	engine.Spawn(chat.New(), "chat", actor.WithID("chat"))
	// starts mdns discovery
	engine.Spawn(discovery.NewMdnsDiscovery(
		discovery.WithAnnounceAddr(*ip, *port),
	), "mdns")

	// Clean exit.
	select {}
}
