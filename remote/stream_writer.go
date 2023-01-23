package remote

import (
	context "context"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type writeStream struct {
	pid *actor.PID
	msg proto.Message
}

type streamWriter struct {
	writeToAddr string
	conn        *grpc.ClientConn
	stream      Remote_ReceiveClient
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
	case writeStream:
		e.handleWriteStream(msg)
	}
}

func (e *streamWriter) init() {
	conn, err := grpc.Dial(e.writeToAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalw("[REMOTE] failed to dial pid", log.M{
			"err":         err,
			"writeToAddr": e.writeToAddr,
		})
	}

	client := NewRemoteClient(conn)
	stream, err := client.Receive(context.Background())
	if err != nil {
		log.Errorw("[REMOTE] streaming receive error", log.M{
			"err":         err,
			"writeToAddr": e.writeToAddr,
		})
	}
	_ = stream
	// e.stream = stream
	e.conn = conn

	log.Debugw("[REMOTE] stream writer started", log.M{
		"writeToAddr": e.writeToAddr,
	})
}

func (e *streamWriter) handleWriteStream(ws writeStream) {
	msg, err := serialize(ws.pid, ws.msg)
	if err != nil {
		log.Errorw("[REMOTE] failed serializing message", log.M{
			"err": err,
		})
	}

	e.stream.Send(msg)
}
