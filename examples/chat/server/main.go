package main

import (
	"flag"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/log"
	"github.com/anthdm/hollywood/remote"
)

type server struct {
	clients map[*actor.PID]string
}

func newServer() actor.Receiver {
	return &server{
		clients: make(map[*actor.PID]string),
	}
}

func (s *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case *types.Message:
		s.handleMessage(ctx, msg)
	case *types.Disconnect:
		username, ok := s.clients[ctx.Sender()]
		if !ok {
			// ignore a non existing client
			return
		}
		delete(s.clients, ctx.Sender())
		log.Infow("client disconnected", log.M{
			"pid":      ctx.Sender(),
			"username": username,
		})
	case *types.Connect:
		s.clients[ctx.Sender()] = msg.Username
		log.Infow("new client connected", log.M{
			"pid":      ctx.Sender(),
			"username": msg.Username,
		})
	}
}

// handle the incoming message by broadcasting it to all connected clients.
func (s *server) handleMessage(ctx *actor.Context, msg *types.Message) {
	for pid := range s.clients {
		// dont send message to ourselves
		if !pid.Equals(ctx.Sender()) {
			ctx.Forward(pid)
		}
	}
}

func main() {
	var (
		listenAt = flag.String("listen", "127.0.0.1:4000", "")
	)
	log.SetLevel(log.LevelInfo)
	e := actor.NewEngine()
	rem := remote.New(e, remote.Config{
		ListenAddr: *listenAt,
	})
	e.WithRemote(rem)
	e.Spawn(newServer, "server")

	select {}
}
