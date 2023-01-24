package remote

import (
	context "context"
	"net"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/protobuf/proto"
	"storj.io/drpc/drpcconn"
)

type writeToStream struct {
	sender *actor.PID
	pid    *actor.PID
	msg    proto.Message
}

type streamWriter struct {
	writeToAddr string
	conn        *drpcconn.Conn
	stream      DRPCRemote_ReceiveStream
}

func newStreamWriter(address string) actor.Producer {
	return func() actor.Receiver {
		return &streamWriter{
			writeToAddr: address,
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
		log.Fatalw("[REMOTE] failed to dial pid", log.M{
			"err":         err,
			"writeToAddr": e.writeToAddr,
		})
	}

	conn := drpcconn.New(rawconn)
	client := NewDRPCRemoteClient(conn)

	stream, err := client.Receive(context.Background())
	if err != nil {
		log.Errorw("[REMOTE] streaming receive error", log.M{
			"err":         err,
			"writeToAddr": e.writeToAddr,
		})
	}

	e.stream = stream
	e.conn = conn

	log.Tracew("[REMOTE] stream writer started", log.M{
		"writeToAddr": e.writeToAddr,
	})
}

func (e *streamWriter) handleWriteStream(ws writeToStream) {
	msg, err := serialize(ws.pid, ws.msg)
	if err != nil {
		log.Errorw("[REMOTE] failed serializing message", log.M{
			"err": err,
		})
	}

	if err := e.stream.Send(msg); err != nil {
		log.Errorw("[REMOTE] failed sending message", log.M{
			"err": err,
		})
	}
}
