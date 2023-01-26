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

type writeToStream struct {
	sender *actor.PID
	pid    *actor.PID
	msg    proto.Message
}

type streamWriter struct {
	writeToAddr string
	rawconn     net.Conn
	conn        *drpcconn.Conn
	stream      DRPCRemote_ReceiveStream
	engine      *actor.Engine
	routerPID   *actor.PID
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
	case writeToStream:
		e.handleWriteStream(msg)
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

func (e *streamWriter) handleWriteStream(ws writeToStream) {
	if e.stream == nil {
		return
	}
	msg, err := serialize(ws.pid, ws.sender, ws.msg)
	if err != nil {
		log.Errorw("[REMOTE] failed serializing message", log.M{
			"err": err,
		})
	}
	if err := e.stream.Send(msg); err != nil {
		if errors.Is(err, io.EOF) {
			e.conn.Close()
			return
		}
		log.Errorw("[REMOTE] failed sending message", log.M{
			"err": err,
		})
	}
	e.rawconn.SetDeadline(time.Now().Add(connIdleTimeout))
}
