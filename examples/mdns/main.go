package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	engine := actor.NewEngine()

	r := remote.New(engine, remote.Config{
		ListenAddr: fmt.Sprintf("%s:%d", *ip, *port),
	})
	engine.WithRemote(r)
	engine.Spawn(chat.New(engine.EventStream), "chat")

	// starts mdns discovery
	engine.Spawn(discovery.NewMdnsDiscovery(
		engine.EventStream,
		discovery.WithAnnounceAddr(*ip, *port),
	), "mdns")

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
