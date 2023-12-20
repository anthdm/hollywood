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
	r := remote.New("127.0.0.1:3000", nil)
	e, err := actor.NewEngine(&actor.EngineOpts{Remote: r})
	if err != nil {
		panic(err)
	}

	pid := actor.NewPID("127.0.0.1:4000", "server")
	for {
		e.Send(pid, &msg.Message{Data: "hello!"})
		slog.Debug("sent message", "to", pid.String())
		time.Sleep(time.Second)
	}
}
