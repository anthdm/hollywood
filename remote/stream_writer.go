package remote

import (
	context "context"
	errors "errors"
	"io"
	"net"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/protobuf/proto"
	"storj.io/drpc/drpcconn"
)

const connIdleTimeout = time.Minute * 10

type streamWriter struct {
	writeToAddr string
	rawconn     net.Conn
	conn        *drpcconn.Conn
	stream      DRPCRemote_ReceiveStream
	engine      *actor.Engine
	routerPID   *actor.PID
	batch       *batch
}

func newStreamWriter(e *actor.Engine, rpid *actor.PID, address string) actor.Producer {
	return func() actor.Receiver {
		return &streamWriter{
			writeToAddr: address,
			engine:      e,
			routerPID:   rpid,
		}
	}
}

func (e *streamWriter) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		e.init()
		e.batch = newBatch(e.send)
	case *streamDeliver:
		e.batch.add(msg)
	}
}

func (e *streamWriter) init() {
	rawconn, err := net.Dial("tcp", e.writeToAddr)
	if err != nil {
		panic(&actor.InternalError{Err: err, From: "[STREAM WRITER]"})
	}
	e.rawconn = rawconn
	rawconn.SetDeadline(time.Now().Add(connIdleTimeout))

	conn := drpcconn.New(rawconn)
	client := NewDRPCRemoteClient(conn)

	stream, err := client.Receive(context.Background())
	if err != nil {
		log.Errorw("[STREAM WRITER] receive error", log.M{
			"err":         err,
			"writeToAddr": e.writeToAddr,
		})
	}

	e.stream = stream
	e.conn = conn

	log.Tracew("[STREAM WRITER] started", log.M{
		"writeToAddr": e.writeToAddr,
	})

	go func() {
		<-e.conn.Closed()
		e.stream.Close()
		e.engine.Send(e.routerPID, terminateStream{address: e.writeToAddr})
	}()
}

func (e *streamWriter) send(streams []*streamDeliver) {
	var (
		typeLookup   = make(map[string]int32)
		typeNames    = make([]string, 0)
		senderLookup = make(map[uint64]int32)
		senders      = make([]*actor.PID, 0)
		targetLookup = make(map[uint64]int32)
		targets      = make([]*actor.PID, 0)
		messages     = make([]*Message, len(streams))
	)

	for i := 0; i < len(streams); i++ {
		var (
			stream   = streams[i]
			typeID   int32
			senderID int32
			targetID int32
		)
		typeID, typeNames = lookupTypeName(typeLookup, string(proto.MessageName(stream.msg)), typeNames)
		senderID, senders = lookupPIDs(senderLookup, stream.sender, senders)
		targetID, targets = lookupPIDs(targetLookup, stream.target, targets)

		b, err := stream.msg.MarshalVT()
		if err != nil {
			panic(err)
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

	if err := e.stream.Send(env); err != nil {
		if errors.Is(err, io.EOF) {
			e.conn.Close()
			return
		}
		log.Errorw("[REMOTE] failed sending message", log.M{
			"err": err,
		})
	}
	// refresh the connection deadline.
	e.rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
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
