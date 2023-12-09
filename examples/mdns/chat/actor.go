package chat

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/mdns/chat/types"
	"github.com/anthdm/hollywood/examples/mdns/discovery"
	"github.com/anthdm/hollywood/log"
)

type server struct {
	eventStream  *actor.EventStream
	subscription *actor.EventSub
	ctx          *actor.Context
}

func New(e *actor.EventStream) actor.Producer {
	return func() actor.Receiver {
		ret := &server{
			eventStream: e,
		}
		return ret
	}
}

func (s *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		s.ctx = ctx
		s.subscription = s.eventStream.Subscribe(s.onMessage)
	case actor.Started:
		_ = msg
	case actor.Stopped:
		s.shutdown()
	case *types.Message:
		s.handleMessage(ctx, msg)
	}
}
func (s *server) onMessage(event any) {
	switch evt := event.(type) {
	case *discovery.DiscoveryEvent:
		pid := actor.NewPID(evt.Addr[0], "chat")
		s.ctx.Engine().Send(pid, &types.Message{
			Username: evt.ID,
			Msg:      "hello",
		})
	}
}

func (s *server) shutdown() {
	s.eventStream.Unsubscribe(s.subscription)
}

// handle the incoming message by broadcasting it to all connected clients.
func (s *server) handleMessage(ctx *actor.Context, msg *types.Message) {
	log.Infow("new message", log.M{"msg": msg.Msg})
}
