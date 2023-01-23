package remote

import (
	"net"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"google.golang.org/grpc"
)

type streamReader struct {
	UnimplementedRemoteServer

	remote *Remote
}

func newStreamReader(r *Remote) *streamReader {
	return &streamReader{
		remote: r,
	}
}

func (r *streamReader) Receive(stream Remote_ReceiveServer) error {
	defer func() {
		log.Warnw("[REMOTE] stream reader terminated", log.M{})
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			if strings.Contains(err.Error(), "Canceled desc") {
				break
			}
			log.Errorw("[REMOTE] stream receive", log.M{"err": err})
			return err
		}

		pid := msg.Target
		dmsg, err := deserialize(msg.Data, msg.TypeName)
		if err != nil {
			log.Errorw("[REMOTE] deserialize", log.M{"err": err})
		}

		apid := actor.NewPID(pid.Address, pid.Name)

		r.remote.engine.Send(apid, dmsg)
	}

	return nil
}

type Config struct {
	ListenAddr string
}

type Remote struct {
	engine       *actor.Engine
	config       Config
	streamReader *streamReader
}

func New(e *actor.Engine, cfg Config) *Remote {
	r := &Remote{
		engine: e,
		config: cfg,
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

func (r *Remote) Address() string {
	return r.config.ListenAddr
}
