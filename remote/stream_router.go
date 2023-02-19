package remote

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
)

type streamDeliver struct {
	sender *actor.PID
	target *actor.PID
	msg    Marshaler
}

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
	case *streamDeliver:
		s.deliverStream(msg)
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

func (s *streamRouter) deliverStream(msg *streamDeliver) {
	var (
		swpid   *actor.PID
		ok      bool
		address = msg.target.Address
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
	s.engine.Send(swpid, msg)
}
