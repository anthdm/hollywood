package remote

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/protobuf/proto"
)

type routeToStream struct {
	pid *actor.PID
	msg proto.Message
}

type terminateStream struct {
	address string
}

type streamRouter struct {
	engine *actor.Engine
	// streams is a map of remote address to stream writer pid.
	streams map[string]*actor.PID
	pid     *actor.PID
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
	case actor.Started:
		s.pid = ctx.PID()
	case routeToStream:
		s.handleRouteToStream(msg)
	case terminateStream:
		s.handleTerminateStream(msg)
	}
}

func (s *streamRouter) handleTerminateStream(msg terminateStream) {
	delete(s.streams, msg.address)
	log.Tracew("[STREAM ROUTER] terminating stream", log.M{
		"stream": msg.address,
	})
}

func (s *streamRouter) handleRouteToStream(msg routeToStream) {
	var (
		swpid *actor.PID
		ok    bool
	)

	address := msg.pid.Address
	swpid, ok = s.streams[address]
	if !ok {
		swpid = s.engine.Spawn(newStreamWriter(s.engine, s.pid, address), "stream", address)
		s.streams[address] = swpid
		log.Tracew("[STREAM ROUTER] new stream route", log.M{
			"route": swpid,
		})
	}

	ws := writeToStream{
		pid: msg.pid,
		msg: msg.msg,
	}

	s.engine.Send(swpid, ws)
}
