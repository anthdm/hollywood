package remote

import (
	"crypto/tls"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

// streamDeliver is a struct that represents a message to be delivered over a stream.
// It contains the sender's PID, the target's PID, and the message itself.
type streamDeliver struct {
	sender *actor.PID
	target *actor.PID
	msg    any
}

// terminateStream is a struct that represents a request to terminate a stream.
// It contains the address of the stream to be terminated.
type terminateStream struct {
	address string
}

// streamRouter is a struct that routes messages to streams.
// It contains an actor engine, a map of streams, a PID, and a TLS configuration.
type streamRouter struct {
	engine *actor.Engine
	// streams is a map of remote address to stream writer pid.
	streams   map[string]*actor.PID
	pid       *actor.PID
	tlsConfig *tls.Config
}

// newStreamRouter is a function that creates a new instance of a streamRouter.
// It takes an actor engine and a TLS configuration as parameters.
// It returns a function that creates a new streamRouter when called.
// This returned function is of type actor.Producer, which is a function that returns an actor.Receiver.
func newStreamRouter(e *actor.Engine, tlsConfig *tls.Config) actor.Producer {
	return func() actor.Receiver {
		// Create a new streamRouter with an empty streams map, the provided engine, and the provided TLS configuration.
		return &streamRouter{
			streams:   make(map[string]*actor.PID),
			engine:    e,
			tlsConfig: tlsConfig,
		}
	}
}

// Receive is a method that handles incoming messages to the streamRouter.
// It takes a context of type *actor.Context.
// Depending on the type of the message, it performs different actions.
func (s *streamRouter) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	// If the message is of type actor.Started, it sets the streamRouter's pid to the context's PID.
	case actor.Started:
		s.pid = ctx.PID()
	// If the message is of type *streamDeliver, it calls the deliverStream method with the message.
	case *streamDeliver:
		s.deliverStream(msg)
	// If the message is of type terminateStream, it calls the handleTerminateStream method with the message.
	case terminateStream:
		s.handleTerminateStream(msg)
	}
}

// handleTerminateStream is a method that handles a terminateStream message.
// It takes a terminateStream message as a parameter.
// It finds the stream writer PID associated with the address in the message, deletes it from the streams map,
// and logs a debug message indicating that the stream is being terminated.
func (s *streamRouter) handleTerminateStream(msg terminateStream) {
	// Get the stream writer PID associated with the address in the message.
	streamWriterPID := s.streams[msg.address]
	// Delete the stream writer PID from the streams map.
	delete(s.streams, msg.address)
	// Log a debug message indicating that the stream is being terminated.
	slog.Debug("terminating stream",
		"remote", msg.address,
		"pid", streamWriterPID,
	)
}

// deliverStream is a method that delivers a streamDeliver message.
// It takes a streamDeliver message as a parameter.
// It checks if there is a stream writer PID associated with the target address in the streams map.
// If there isn't, it creates a new stream writer, adds it to the streams map, and sends the message to it.
// If there is, it simply sends the message to the existing stream writer.
func (s *streamRouter) deliverStream(msg *streamDeliver) {
	var (
		swpid   *actor.PID
		ok      bool
		address = msg.target.Address
	)

	// Check if there is a stream writer PID associated with the target address in the streams map.
	swpid, ok = s.streams[address]
	if !ok {
		// If there isn't, create a new stream writer and add it to the streams map.
		swpid = s.engine.SpawnProc(newStreamWriter(s.engine, s.pid, address, s.tlsConfig))
		s.streams[address] = swpid
	}

	// Send the message to the stream writer.
	s.engine.Send(swpid, msg)
}
