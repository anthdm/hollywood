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

func (f *server) Receive(ctx *actor.Context) {
	switch m := ctx.Message().(type) {
	case actor.Started:
		slog.Info("server started")
		fmt.Println("server has started")
	case *actor.PID:
		slog.Info("server got pid", "pid", m)
	case *msg.Message:
		slog.Info("server got message", "msg", m)
	default:
		slog.Warn("server got unknown message", "msg", m, "type", reflect.TypeOf(m).String())
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	r := remote.New("127.0.0.1:4000", nil)
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(r))
	if err != nil {
		panic(err)
	}

	e.Spawn(newServer, "server", actor.WithID("primary"))
	select {}
}
