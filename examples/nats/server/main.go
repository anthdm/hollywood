package main

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/remote/msg"
	"github.com/anthdm/hollywood/remote"
)

type server struct{}

func newServer() actor.Receiver {
	return &server{}
}

func (s *server) Receive(ctx *actor.Context) {
	switch m := ctx.Message().(type) {
	case actor.Started:
		slog.Info("nats server started")
		fmt.Println("nats server has started")
	case *actor.PID:
		slog.Info("nats server got pid", "pid", m)
	case *msg.Message:
		slog.Info("nats server got message", "msg", m)
	default:
		slog.Warn("nats server got unknown message", "msg", m, "type", reflect.TypeOf(m).String())
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Uses NATS default URL (nats://127.0.0.1:4222) when no URL is configured.
	r := remote.NewNats("nats-server-node", remote.NatsConfig{})
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(r))
	if err != nil {
		panic(err)
	}

	e.Spawn(newServer, "server", actor.WithID("primary"))
	select {}
}
