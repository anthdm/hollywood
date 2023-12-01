package main

import (
	"flag"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/remote"
)

type server struct {
	clients map[*actor.PID]string
	logger  *slog.Logger
}

func newServer() actor.Receiver {
	return &server{
		clients: make(map[*actor.PID]string),
		logger:  slog.Default(),
	}
}

func (s *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case *types.Message:
		s.handleMessage(ctx, msg)
	case *types.Disconnect:
		s.logger.Info("client disconnected", "pid", ctx.Sender())
		username, ok := s.clients[ctx.Sender()]
		if !ok {
			s.logger.Warn("unknown client disconnected", "client", ctx.Sender().GetID())
			return
		}
		delete(s.clients, ctx.Sender())
		slog.Info("client disconnected",
			"pid", ctx.Sender(),
			"username", username)
	case *types.Connect:
		s.clients[ctx.Sender()] = msg.Username
		slog.Info("new client connected",
			"pid", ctx.Sender(),
			"username", msg.Username,
		)
	}
}

// handle the incoming message by broadcasting it to all connected clients.
func (s *server) handleMessage(ctx *actor.Context, msg *types.Message) {
	s.logger.Info("incoming message", "from", msg.Username, "message", msg.Msg)
	for pid := range s.clients {
		// dont send message to the place where it came from.
		if !pid.Equals(ctx.Sender()) {
			s.logger.Info("forwarding message", "pid", pid.ID, "addr", pid.Address)
			ctx.Forward(pid)
		}
	}
}

func main() {
	var (
		listenAt = flag.String("listen", "127.0.0.1:4000", "")
	)
	flag.Parse()
	e := actor.NewEngine()
	rem := remote.New(e, remote.Config{
		ListenAddr: *listenAt,
	})
	e.WithRemote(rem)
	e.Spawn(newServer, "server")

	select {}
}
