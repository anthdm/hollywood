package remote

import (
	"context"
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
	connIdleTimeout       = time.Minute * 10
	streamWriterBatchSize = 1024
)

type streamWriter struct {
	writeToAddr string
	rawconn     net.Conn
	conn        *drpcconn.Conn
	stream      DRPCRemote_ReceiveStream
	engine      *actor.Engine
	routerPID   *actor.PID
	pid         *actor.PID
	inbox       actor.Inboxer
	serializer  Serializer
}

func newStreamWriter(e *actor.Engine, rpid *actor.PID, address string) actor.Processer {
	return &streamWriter{
		writeToAddr: address,
		engine:      e,
		routerPID:   rpid,
		inbox:       actor.NewInbox(streamWriterBatchSize),
		pid:         actor.NewPID(e.Address(), "stream", address),
		serializer:  ProtoSerializer{},
	}
}

func (s *streamWriter) PID() *actor.PID { return s.pid }
func (s *streamWriter) Send(_ *actor.PID, msg any, sender *actor.PID) {
	s.inbox.Send(actor.Envelope{Msg: msg, Sender: sender})
}

func (s *streamWriter) Invoke(msgs []actor.Envelope) {
	var (
		typeLookup   = make(map[string]int32)
		typeNames    = make([]string, 0)
		senderLookup = make(map[uint64]int32)
		senders      = make([]*actor.PID, 0)
		targetLookup = make(map[uint64]int32)
		targets      = make([]*actor.PID, 0)
		messages     = make([]*Message, len(msgs))
	)

	for i := 0; i < len(msgs); i++ {
		var (
			stream   = msgs[i].Msg.(*streamDeliver)
			typeID   int32
			senderID int32
			targetID int32
		)
		typeID, typeNames = lookupTypeName(typeLookup, s.serializer.TypeName(stream.msg), typeNames)
		senderID, senders = lookupPIDs(senderLookup, stream.sender, senders)
		targetID, targets = lookupPIDs(targetLookup, stream.target, targets)

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

	env := &Envelope{
		Senders:   senders,
		Targets:   targets,
		TypeNames: typeNames,
		Messages:  messages,
	}

	if err := s.stream.Send(env); err != nil {
		if errors.Is(err, io.EOF) {
			_ = s.conn.Close()
			return
		}
		slog.Error("stream writer failed sending message",
			"err", err,
		)
	}
	// refresh the connection deadline.
	err := s.rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
	if err != nil {
		slog.Error("failed to set context deadline", "err", err)
	}
}

func (s *streamWriter) init() {
	var (
		rawconn net.Conn
		err     error
		delay   time.Duration = time.Millisecond * 500
	)
	for {
		rawconn, err = net.Dial("tcp", s.writeToAddr)
		if err != nil {
			slog.Error("net.Dial", "err", err, "remote", s.writeToAddr)
			time.Sleep(delay)
			continue
		}
		break
	}
	if rawconn == nil {
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

func (s *streamWriter) Shutdown(wg *sync.WaitGroup) {
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

func (s *streamWriter) Start() {
	s.inbox.Start(s)
	s.init()
}

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
