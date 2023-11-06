package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"storj.io/drpc/drpcconn"
)

const (
	connIdleTimeout       = time.Minute * 10
	streamWriterBatchSize = 1024 * 32
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
	logger      log.Logger
}

func newStreamWriter(e *actor.Engine, rpid *actor.PID, address string, logger log.Logger) actor.Processer {
	return &streamWriter{
		writeToAddr: address,
		engine:      e,
		routerPID:   rpid,
		inbox:       actor.NewInbox(streamWriterBatchSize),
		pid:         actor.NewPID(e.Address(), "stream", address),
		serializer:  ProtoSerializer{},
		logger:      logger,
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
			s.logger.Errorw("serialize", "err", err)
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
			s.conn.Close()
			return
		}
		s.logger.Errorw("failed sending message",
			"err", err,
		)
	}
	// refresh the connection deadline.
	s.rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
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
			s.logger.Errorw("net.Dial", "err", err, "remote", s.writeToAddr)
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
	rawconn.SetDeadline(time.Now().Add(connIdleTimeout))

	conn := drpcconn.New(rawconn)
	client := NewDRPCRemoteClient(conn)

	stream, err := client.Receive(context.Background())
	if err != nil {
		s.logger.Errorw("receive", "err", err, "remote", s.writeToAddr)
	}

	s.stream = stream
	s.conn = conn

	s.logger.Debugw("connected",
		"remote", s.writeToAddr,
	)

	go func() {
		<-s.conn.Closed()
		s.logger.Debugw("lost connection",
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
