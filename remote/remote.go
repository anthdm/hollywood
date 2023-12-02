package remote

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

type Config struct {
	ListenAddr string
	Wg         *sync.WaitGroup
}

type Remote struct {
	engine          *actor.Engine
	config          Config
	streamReader    *streamReader
	streamRouterPID *actor.PID
	logger          log.Logger
	stopCh          chan struct{} // Stop closes this channel to signal the remote to stop listening.
	stopWg          *sync.WaitGroup
}

// New creates a new "Remote" object given an engine and a Config.
func New(e *actor.Engine, cfg Config) *Remote {
	r := &Remote{
		engine: e,
		config: cfg,
		logger: e.GetLogger().SubLogger("[remote]"),
	}
	r.streamReader = newStreamReader(r)
	return r
}

// Start starts the remote. The remote will listen on the given address.
// It will spawn a new actor, "router", which will handle all incoming messages.
// When the supplied context is cancelled, the remote will stop listening.
func (r *Remote) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.stopCh = make(chan struct{})
	ln, err := net.Listen("tcp", r.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	r.logger.Debugw("listening", "addr", r.config.ListenAddr)
	mux := drpcmux.New()
	err = DRPCRegisterRemote(mux, r.streamReader)
	if err != nil {
		return fmt.Errorf("failed to register remote: %w", err)
	}
	s := drpcserver.New(mux)
	r.streamRouterPID = r.engine.Spawn(newStreamRouter(r.engine, r.logger), "router", actor.WithInboxSize(1024*1024))
	r.logger.Infow("server started", "listenAddr", r.config.ListenAddr)
	// start the server in a goroutine
	r.stopWg = &sync.WaitGroup{}
	r.stopWg.Add(1)
	go func() {
		defer r.stopWg.Done()
		err := s.Serve(ctx, ln)
		if err != nil {
			r.logger.Errorw("drpcserver", "err", err)
		} else {
			r.logger.Infow("drpcserver stopped")
		}
	}()
	// wait for stopCh to be closed
	go func() {
		<-r.stopCh
		r.logger.Debugw("cancelling context")
		cancel()
	}()
	return nil
}

func (r *Remote) Stop() *sync.WaitGroup {
	r.stopCh <- struct{}{}
	return r.stopWg
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
