package remote

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/anthdm/hollywood/actor"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

// Config holds the remote configuration.
type Config struct {
	TLSConfig *tls.Config
	// Wg        *sync.WaitGroup
}

// NewConfig returns a new default remote configuration.
func NewConfig() Config {
	return Config{}
}

// WithTLS sets the TLS config of the remote which will set
// the transport of the Remote to TLS.
func (c Config) WithTLS(tlsconf *tls.Config) Config {
	c.TLSConfig = tlsconf
	return c
}

type Remote struct {
	addr            string
	engine          *actor.Engine
	config          Config
	streamReader    *streamReader
	streamRouterPID *actor.PID
	stopCh          chan struct{} // Stop closes this channel to signal the remote to stop listening.
	stopWg          *sync.WaitGroup
	state           atomic.Uint32
}

const (
	stateInvalid uint32 = iota
	stateInitialized
	stateRunning
	stateStopped
)

// New creates a new "Remote" object given a Config.
func New(addr string, config Config) *Remote {
	r := &Remote{
		addr:   addr,
		config: config,
	}
	r.state.Store(stateInitialized)
	r.streamReader = newStreamReader(r)
	return r
}

func (r *Remote) Start(e *actor.Engine) error {
	if r.state.Load() != stateInitialized {
		return fmt.Errorf("remote already started")
	}
	r.state.Store(stateRunning)
	r.engine = e
	var ln net.Listener
	var err error
	switch r.config.TLSConfig {
	case nil:
		ln, err = net.Listen("tcp", r.addr)
	default:
		slog.Debug("remote using TLS for listening")
		ln, err = tls.Listen("tcp", r.addr, r.config.TLSConfig)
	}
	if err != nil {
		return fmt.Errorf("remote failed to listen: %w", err)
	}
	slog.Debug("listening", "addr", r.addr)
	mux := drpcmux.New()
	err = DRPCRegisterRemote(mux, r.streamReader)
	if err != nil {
		return fmt.Errorf("failed to register remote: %w", err)
	}
	s := drpcserver.New(mux)

	r.streamRouterPID = r.engine.Spawn(
		newStreamRouter(r.engine, r.config.TLSConfig),
		"router", actor.WithInboxSize(1024*1024))
	slog.Debug("server started", "listenAddr", r.addr)
	r.stopWg = &sync.WaitGroup{}
	r.stopWg.Add(1)
	r.stopCh = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer r.stopWg.Done()
		err := s.Serve(ctx, ln)
		if err != nil {
			slog.Error("drpcserver", "err", err)
		} else {
			slog.Debug("drpcserver stopped")
		}
	}()
	// wait for stopCh to be closed
	go func() {
		<-r.stopCh
		cancel()
	}()
	return nil
}

// Stop will stop the remote from listening.
func (r *Remote) Stop() *sync.WaitGroup {
	if r.state.Load() != stateRunning {
		slog.Warn("remote already stopped but stop was called", "state", r.state.Load())
		return &sync.WaitGroup{} // return empty waitgroup so the caller can still wait without panicking.
	}
	r.state.Store(stateStopped)
	r.stopCh <- struct{}{}
	return r.stopWg
}

// Send sends the given message to the process with the given pid over the network.
// Optional a "Sender PID" can be given to inform the receiving process who sent the
// message.
// Sending will work even if the remote is stopped. Receiving however, will not work.
func (r *Remote) Send(pid *actor.PID, msg any, sender *actor.PID) {
	r.engine.Send(r.streamRouterPID, &streamDeliver{
		target: pid,
		sender: sender,
		msg:    msg,
	})
}

// Address returns the listen address of the remote.
func (r *Remote) Address() string {
	return r.addr
}

func init() {
	RegisterType(&actor.PID{})
}
