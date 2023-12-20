package chat

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/mdns/chat/types"
	"github.com/anthdm/hollywood/examples/mdns/discovery"
	"log/slog"
	"reflect"
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
	case *discovery.DiscoveryEvent:
		// a new remote actor has been discovered. We can now send messages to it.
		pid := actor.NewPID(msg.Addr[0], "chat")
		chatMsg := &types.Message{
			Username: "helloer",
			Msg:      "hello there",
		}
		slog.Info("sending hello", "to", pid.String(), "msg", chatMsg.Msg, "from", ctx.PID().String())
		s.engine.SendWithSender(pid, chatMsg, ctx.PID())
	case *types.Message:
		s.handleMessage(ctx, msg)
	case actor.DeadLetterEvent:
		slog.Warn("dead letter", "sender", msg.Sender, "target", msg.Target, "msg", msg.Message)
	default:
		slog.Warn("unknown message", "type", reflect.TypeOf(msg).String(), "msg", msg)
	}
}

// handle the incoming message by broadcasting it to all connected clients.
func (s *server) handleMessage(_ *actor.Context, msg *types.Message) {
	slog.Info("new message", "msg", msg.Msg)
}
