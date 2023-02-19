package remote

import (
	context "context"
	errors "errors"
	fmt "fmt"
	"io"
	"net"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	proto "google.golang.org/protobuf/proto"
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
	env, err := makeEnvelope(streams)
	if err != nil {
		log.Errorw("[STREAM WRITER] failed creating message envelope", log.M{
			"err": err,
		})
		return
	}
	fmt.Println("env len", len(env.Messages))
	fmt.Println("env total size", proto.Size(env))
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
