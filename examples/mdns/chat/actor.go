package chat

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/mdns/chat/types"
	"github.com/anthdm/hollywood/examples/mdns/discovery"
	"log/slog"
)

type server struct {
	engine *actor.Engine
}

func New() actor.Producer {
	return func() actor.Receiver {
		ret := &server{}
		return ret
	}
}

func (s *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		s.engine = ctx.Engine()
		ctx.Engine().Subscribe(ctx.PID())
	case actor.Started:
		_ = msg
	case actor.Stopped:
		ctx.Engine().Unsubscribe(ctx.PID())
	case *types.Message:
		s.handleMessage(ctx, msg)
	}
}
func (s *server) onMessage(event any) {
	switch evt := event.(type) {
	case *discovery.DiscoveryEvent:
		pid := actor.NewPID(evt.Addr[0], "chat")
		s.engine.Send(pid, &types.Message{
			Username: evt.ID,
			Msg:      "hello",
		})
	}
}

// handle the incoming message by broadcasting it to all connected clients.
func (s *server) handleMessage(_ *actor.Context, msg *types.Message) {
	slog.Info("new message", "msg", msg.Msg)
}
