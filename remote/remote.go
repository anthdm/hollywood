package remote

import (
	"context"
	"net"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

// Config holds the remote configuration.
type Config struct {
	ListenAddr string
	Logger     log.Logger
}

type Remote struct {
	engine          *actor.Engine
	config          Config
	streamReader    *streamReader
	streamRouterPID *actor.PID
	logger          log.Logger
}

// New creates a new "Remote" object given an engine and a Config.
func New(e *actor.Engine, cfg Config) *Remote {
	r := &Remote{
		engine: e,
		config: cfg,
		logger: cfg.Logger,
	}
	r.streamReader = newStreamReader(r)
	return r
}

func (r *Remote) Start() {
	ln, err := net.Listen("tcp", r.config.ListenAddr)
	if err != nil {
		panic("failed to listen: " + err.Error())
	}

	mux := drpcmux.New()
	err = DRPCRegisterRemote(mux, r.streamReader)
	if err != nil {
		r.logger.Errorw("failed to register remote", "err", err)
	}
	s := drpcserver.New(mux)

	r.streamRouterPID = r.engine.Spawn(newStreamRouter(r.engine, r.logger), "router", actor.WithInboxSize(1024*1024))
	r.logger.Infow("server started", "listenAddr", r.config.ListenAddr)
	ctx := context.Background()
	go func() {
		err := s.Serve(ctx, ln)
		r.logger.Errorw("drpcserver stopped", "err", err)
	}()
}

// Send sends the given message to the process with the given pid over the network.
// Optional a "Sender PID" can be given to inform the receiving process who sent the
// message.
func (r *Remote) Send(pid *actor.PID, msg any, sender *actor.PID) {
	r.engine.Send(r.streamRouterPID, &streamDeliver{
		target: pid,
		sender: sender,
		msg:    msg,
	})
}

// Address returns the listen address of the remote.
func (r *Remote) Address() string {
	return r.config.ListenAddr
}

func init() {
	RegisterType(&actor.PID{})
}
