package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"

	"github.com/fancom/hollywood/actor"
	"github.com/fancom/hollywood/examples/mdns/chat"
	"github.com/fancom/hollywood/examples/mdns/discovery"
	"github.com/fancom/hollywood/remote"
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
	rem := remote.New(fmt.Sprintf("%s:%d", *ip, *port), remote.NewConfig())
	engine, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(rem))
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
