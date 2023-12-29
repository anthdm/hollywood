package remote

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"storj.io/drpc/drpcconn"
)

const (
	// connIdleTimeout is the duration after which the connection will be closed if idle.
	connIdleTimeout = time.Minute * 10
	// streamWriterBatchSize is the size of the batch that the stream writer can handle.
	streamWriterBatchSize = 1024
)

// streamWriter is a struct that handles writing to a stream.
type streamWriter struct {
	// writeToAddr is the address to which the stream writer will write.
	writeToAddr string
	// rawconn is the raw network connection used by the stream writer.
	rawconn net.Conn
	// conn is the DRPC connection used by the stream writer.
	conn *drpcconn.Conn
	// stream is the DRPC stream that the stream writer will write to.
	stream DRPCRemote_ReceiveStream
	// engine is the actor engine that powers the stream writer.
	engine *actor.Engine
	// routerPID is the PID of the router actor.
	routerPID *actor.PID
	// pid is the PID of the stream writer itself.
	pid *actor.PID
	// inbox is the inbox that the stream writer uses to receive messages.
	inbox actor.Inboxer
	// serializer is the serializer used to serialize messages before they are written to the stream.
	serializer Serializer
	// tlsConfig is the TLS configuration used for secure connections.
	tlsConfig *tls.Config
}

// newStreamWriter is a function that creates a new instance of a streamWriter.
// It takes an actor engine, a router PID, an address string, and a TLS configuration as parameters.
// It returns an actor.Processer, which is an interface that the streamWriter struct implements.
func newStreamWriter(e *actor.Engine, rpid *actor.PID, address string, tlsConfig *tls.Config) actor.Processer {
	return &streamWriter{
		writeToAddr: address,
		engine:      e,
		routerPID:   rpid,
		inbox:       actor.NewInbox(streamWriterBatchSize),
		pid:         actor.NewPID(e.Address(), "stream"+"/"+address),
		serializer:  ProtoSerializer{},
		tlsConfig:   tlsConfig,
	}
}

// PID is a method that returns the PID (Process ID) of the streamWriter.
func (s *streamWriter) PID() *actor.PID { return s.pid }

// Send is a method that sends a message to the streamWriter's inbox.
// It takes a sender PID and a message as parameters.
// The sender PID is the PID of the actor that sends the message.
// The message is wrapped in an actor.Envelope before it is sent to the inbox.
func (s *streamWriter) Send(_ *actor.PID, msg any, sender *actor.PID) {
	s.inbox.Send(actor.Envelope{Msg: msg, Sender: sender})
}

// Invoke is a method that processes a batch of messages encapsulated in actor.Envelope.
// It serializes the messages, wraps them in a new Envelope, and sends them over the stream.
func (s *streamWriter) Invoke(msgs []actor.Envelope) {
	// Initialize lookup maps and slices for type names, senders, targets, and messages.
	var (
		typeLookup   = make(map[string]int32)
		typeNames    = make([]string, 0)
		senderLookup = make(map[uint64]int32)
		senders      = make([]*actor.PID, 0)
		targetLookup = make(map[uint64]int32)
		targets      = make([]*actor.PID, 0)
		messages     = make([]*Message, len(msgs))
	)

	// Iterate over the messages.
	for i := 0; i < len(msgs); i++ {
		var (
			stream   = msgs[i].Msg.(*streamDeliver)
			typeID   int32
			senderID int32
			targetID int32
		)
		// Lookup or add new type name, sender, and target.
		typeID, typeNames = lookupTypeName(typeLookup, s.serializer.TypeName(stream.msg), typeNames)
		senderID, senders = lookupPIDs(senderLookup, stream.sender, senders)
		targetID, targets = lookupPIDs(targetLookup, stream.target, targets)

		// Serialize the message.
		b, err := s.serializer.Serialize(stream.msg)
		if err != nil {
			slog.Error("serialize", "err", err)
			continue
		}

		messages[i] = &Message{
			Data:          b,
			TypeNameIndex: typeID,
			SenderIndex:   senderID,
			TargetIndex:   targetID,
		}
	}

	// Create a new Envelope with the senders, targets, type names, and messages.
	env := &Envelope{
		Senders:   senders,
		Targets:   targets,
		TypeNames: typeNames,
		Messages:  messages,
	}

	// Send the Envelope over the stream.
	if err := s.stream.Send(env); err != nil {
		if errors.Is(err, io.EOF) {
			_ = s.conn.Close()
			return
		}
		slog.Error("stream writer failed sending message",
			"err", err,
		)
	}
	// Refresh the connection deadline.
	err := s.rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
	if err != nil {
		slog.Error("failed to set context deadline", "err", err)
	}
}

// init is a method that initializes the streamWriter.
// It tries to establish a connection to the remote address.
// If it fails to connect after a certain number of retries, it shuts down the streamWriter and broadcasts a RemoteUnreachableEvent.
func (s *streamWriter) init() {
	var (
		rawconn    net.Conn
		err        error
		delay      time.Duration = time.Millisecond * 500
		maxRetries               = 3
	)
	for i := 0; i < maxRetries; i++ {
		// Here we try to connect to the remote address.
		// @TODO: can we make an Event here in case of failure?
		switch s.tlsConfig {
		case nil:
			rawconn, err = net.Dial("tcp", s.writeToAddr)
			if err != nil {
				d := time.Duration(delay * time.Duration(i*2))
				slog.Error("net.Dial", "err", err, "remote", s.writeToAddr, "retry", i, "max", maxRetries, "delay", d)
				time.Sleep(d)
				continue
			}
		default:
			slog.Debug("remote using TLS for writing")
			rawconn, err = tls.Dial("tcp", s.writeToAddr, s.tlsConfig)
			if err != nil {
				d := time.Duration(delay * time.Duration(i*2))
				slog.Error("tls.Dial", "err", err, "remote", s.writeToAddr, "retry", i, "max", maxRetries, "delay", d)
				time.Sleep(d)
				continue
			}
		}
		break
	}
	// We could not reach the remote after retrying N times. Hence, shutdown the stream writer.
	// and notify RemoteUnreachableEvent.
	if rawconn == nil {
		evt := actor.RemoteUnreachableEvent{
			ListenAddr: s.writeToAddr,
		}
		s.engine.BroadcastEvent(evt)
		s.Shutdown(nil)
		return
	}

	s.rawconn = rawconn
	err = rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
	if err != nil {
		slog.Error("failed to set deadline on raw connection", "err", err)
		return
	}

	conn := drpcconn.New(rawconn)
	client := NewDRPCRemoteClient(conn)

	stream, err := client.Receive(context.Background())
	if err != nil {
		slog.Error("receive", "err", err, "remote", s.writeToAddr)
		s.Shutdown(nil)
		return
	}

	s.stream = stream
	s.conn = conn

	slog.Debug("connected",
		"remote", s.writeToAddr,
	)

	go func() {
		<-s.conn.Closed()
		slog.Debug("lost connection",
			"remote", s.writeToAddr,
		)
		s.Shutdown(nil)
	}()
}

// Shutdown is a method that gracefully shuts down the streamWriter.
// It sends a terminateStream message to the router, closes the stream if it's not nil,
// stops the inbox, removes the streamWriter from the engine's registry, and signals the wait group if it's not nil.
func (s *streamWriter) Shutdown(wg *sync.WaitGroup) {
	// Send a terminateStream message to the router.
	s.engine.Send(s.routerPID, terminateStream{address: s.writeToAddr})
	if s.stream != nil {
		s.stream.Close()
	}
	s.inbox.Stop()
	s.engine.Registry.Remove(s.PID())
	if wg != nil {
		wg.Done()
	}
}

// Start is a method that starts the streamWriter.
// It starts the inbox and initializes the streamWriter.
func (s *streamWriter) Start() {
	s.inbox.Start(s)
	s.init()
}

// lookupPIDs is a function that checks if a PID is in the map.
// If the PID is not in the map, it adds the PID to the map and the slice, and returns the new ID and the updated slice.
// If the PID is in the map, it returns the existing ID and the original slice.
func lookupPIDs(m map[uint64]int32, pid *actor.PID, pids []*actor.PID) (int32, []*actor.PID) {
	if pid == nil {
		return 0, pids
	}
	max := int32(len(m))
	key := pid.LookupKey()
	id, ok := m[key]
	if !ok {
		m[key] = max
		id = max
		pids = append(pids, pid)
	}
	return id, pids

}

// lookupTypeName is a function that checks if a type name is in the map.
// If the type name is not in the map, it adds the type name to the map and the slice, and returns the new ID and the updated slice.
// If the type name is in the map, it returns the existing ID and the original slice.
func lookupTypeName(m map[string]int32, name string, types []string) (int32, []string) {
	max := int32(len(m))
	id, ok := m[name]
	if !ok {
		m[name] = max
		id = max
		types = append(types, name)
	}
	return id, types
}
