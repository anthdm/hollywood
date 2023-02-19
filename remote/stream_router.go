package remote

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
)

type routeToStream struct {
	sender *actor.PID
	pid    *actor.PID
	msg    Marshaler
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
	streamWriterPID := s.streams[msg.address]
	s.engine.Poison(streamWriterPID)
	delete(s.streams, msg.address)
	log.Tracew("[STREAM ROUTER] terminating stream", log.M{
		"dest": msg.address,
		"pid":  streamWriterPID,
	})
}

func (s *streamRouter) handleRouteToStream(msg routeToStream) {
	var (
		swpid   *actor.PID
		ok      bool
		address = msg.pid.Address
	)

	swpid, ok = s.streams[address]
	if !ok {
		swpid = s.engine.Spawn(
			newStreamWriter(s.engine, s.pid, address),
			"stream", actor.WithTags(address), actor.WithInboxSize(1024*1024))
		s.streams[address] = swpid
		log.Tracew("[STREAM ROUTER] new stream route", log.M{
			"pid": swpid,
		})
	}
	ws := writeToStream{
		pid:    msg.pid,
		msg:    msg.msg,
		sender: msg.sender,
	}
	s.engine.Send(swpid, ws)
}
