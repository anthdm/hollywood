package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Uses NATS default URL (nats://127.0.0.1:4222) when no URL is configured.
	r := remote.NewNats("nats-client-node", remote.NatsConfig{})
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(r))
	if err != nil {
		panic(err)
	}

	serverPID := actor.NewPID("nats-server-node", "server/primary")
	// The server will be started with id "primary" on node "nats-server-node".
	for {
		e.Send(serverPID, &msg.Message{Data: "hello over NATS!"})
		slog.Debug("sent message", "to", serverPID.String())
		time.Sleep(time.Second)
	}
}
