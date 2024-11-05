package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/fancom/hollywood/actor"
	"github.com/fancom/hollywood/examples/remote/msg"
	"github.com/fancom/hollywood/remote"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	r := remote.New("127.0.0.1:3000", remote.NewConfig())
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(r))
	if err != nil {
		panic(err)
	}

	serverPID := actor.NewPID("127.0.0.1:4000", "server/primary")
	// The server will be started with id "primary". Hence, let's create
	// the correct PID for it so its reachable.
	for {
		e.Send(serverPID, &msg.Message{Data: "hello!"})
		slog.Debug("sent message", "to", serverPID.String())
		time.Sleep(time.Second)
	}
}
