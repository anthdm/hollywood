package remote

import (
	"context"
	"net"
	"reflect"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

type Config struct {
	ListenAddr string
}

type Remote struct {
	engine          *actor.Engine
	config          Config
	streamReader    *streamReader
	streamRouterPID *actor.PID
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

	mux := drpcmux.New()
	DRPCRegisterRemote(mux, r.streamReader)
	s := drpcserver.New(mux)

	r.streamRouterPID = r.engine.Spawn(newStreamRouter(r.engine), "router", actor.WithInboxSize(1024*1024))

	log.Infow("[REMOTE] server started", log.M{
		"listenAddr": r.config.ListenAddr,
	})

	ctx := context.Background()
	go s.Serve(ctx, ln)
}

func (r *Remote) Send(pid *actor.PID, msg any, sender *actor.PID) {
	switch m := msg.(type) {
	case Marshaler:
		r.engine.Send(r.streamRouterPID, routeToStream{
			pid:    pid,
			msg:    m,
			sender: sender,
		})
	default:
		log.Errorw("[REMOTE] failed to send message", log.M{
			"error": "given message is not of type proto.Message or WithSender",
			"type":  reflect.TypeOf(m),
		})
	}
}

// Address returns the listen address of the remote.
func (r *Remote) Address() string {
	return r.config.ListenAddr
}

func init() {
	RegisterType(&actor.PID{})
}
