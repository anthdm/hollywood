package remote

import (
	"github.com/anthdm/hollywood/actor"
	"google.golang.org/protobuf/proto"
)

type routeToStream struct {
	pid *actor.PID
	msg proto.Message
}

type streamRouter struct {
	engine *actor.Engine
	// streams is a map of remote address to stream writer pid.
	streams map[string]*actor.PID
}

func newStreamRouter(e *actor.Engine) actor.Producer {
	return func() actor.Receiver {
		return &streamRouter{
			streams: make(map[string]*actor.PID),
			engine:  e,
		}
	}
}

func (s *streamRouter) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case routeToStream:
		s.handleRouteToStream(msg)
	}
}

func (s *streamRouter) handleRouteToStream(msg routeToStream) {
	var (
		swpid *actor.PID
		ok    bool
	)

	address := msg.pid.Address
	swpid, ok = s.streams[address]
	if !ok {
		swpid = s.engine.Spawn(newStreamWriter(address), "stream", address)
		s.streams[address] = swpid
	}

	ws := writeToStream{
		pid: msg.pid,
		msg: msg.msg,
	}

	s.engine.Send(swpid, ws)
}
