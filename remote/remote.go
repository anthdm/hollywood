package remote

import (
	"net"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Config struct {
	ListenAddr string
}

type Remote struct {
	engine       *actor.Engine
	config       Config
	streamReader *streamReader
	streams      map[string]*actor.PID
}

func New(e *actor.Engine, cfg Config) *Remote {
	r := &Remote{
		engine:  e,
		config:  cfg,
		streams: make(map[string]*actor.PID),
	}
	r.streamReader = newStreamReader(r)
	return r
}

func (r *Remote) Start() {
	ln, err := net.Listen("tcp", r.config.ListenAddr)
	if err != nil {
		log.Fatalw("[REMOTE] listen", log.M{"err": err})
	}

	grpcserver := grpc.NewServer()
	RegisterRemoteServer(grpcserver, r.streamReader)

	log.Infow("[REMOTE] server started", log.M{
		"listenAddr": r.config.ListenAddr,
	})

	grpcserver.Serve(ln)
}

func (r *Remote) Send(pid *actor.PID, msg any) {
	m, ok := msg.(proto.Message)
	if !ok {
		log.Errorw("[REMOTE] failed to send message", log.M{
			"error": "given message is not of type proto.Message",
		})
		return
	}

	var swpid *actor.PID
	swpid, ok = r.streams[pid.String()]
	if !ok {
		swpid = r.engine.Spawn(newStreamWriter(pid.Address), "stream", pid.Address)
		r.streams[pid.String()] = swpid
	}
	ws := writeStream{
		pid: pid,
		msg: m,
	}
	r.engine.Send(swpid, ws)
}

func (r *Remote) Address() string {
	return r.config.ListenAddr
}
