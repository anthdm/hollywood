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

// Config is a struct that holds the configuration for a Remote.
// It contains a TLS configuration and a WaitGroup.
type Config struct {
	TlsConfig *tls.Config
	Wg        *sync.WaitGroup
}

// Remote is a struct that represents a remote actor system.
// It contains an address, a TLS configuration, a WaitGroup, an actor engine, a Config, a streamReader, a streamRouter PID, a stop channel, a stop WaitGroup, and a state.
type Remote struct {
	addr            string          // The address that the remote system is listening on.
	tlsConfig       *tls.Config     // The TLS configuration for secure communication.
	wg              *sync.WaitGroup // A WaitGroup for synchronizing goroutines.
	engine          *actor.Engine   // The actor engine that powers the remote system.
	config          Config          // The configuration for the remote system.
	streamReader    *streamReader   // The streamReader for reading from streams.
	streamRouterPID *actor.PID      // The PID of the streamRouter.
	stopCh          chan struct{}   // A channel that is closed to signal the remote system to stop listening.
	stopWg          *sync.WaitGroup // A WaitGroup for synchronizing the stopping of the remote system.
	state           atomic.Uint32   // The state of the remote system.
}

// Constants representing the different states a Remote can be in.
const (
	stateInvalid     uint32 = iota // The initial state of a Remote.
	stateInitialized               // The state of a Remote after it has been initialized.
	stateRunning                   // The state of a Remote while it is running.
	stateStopped                   // The state of a Remote after it has been stopped.
)

// New is a function that creates a new Remote.
// It takes an address and a Config as parameters and returns a pointer to a Remote.
// The returned Remote has its address field set to the provided address, its tlsConfig and wg fields set to the ones in the provided Config (if it's not nil),
// its state set to stateInitialized, and its streamReader field set to a new streamReader.
func New(addr string, cfg *Config) *Remote {
	r := &Remote{
		addr: addr,
	}
	if cfg != nil {
		r.tlsConfig = cfg.TlsConfig
		r.wg = cfg.Wg
	}
	r.state.Store(stateInitialized)
	r.streamReader = newStreamReader(r)
	return r
}

// Start is a method that starts the Remote.
// It takes an actor.Engine as a parameter and returns an error.
// It sets the state of the Remote to stateRunning, sets the engine field to the provided Engine,
// listens on the address of the Remote, registers the Remote with the DRPC server, and starts the server.
// It also spawns a new streamRouter and starts a goroutine that serves the DRPC server until the stopCh channel is closed.
func (r *Remote) Start(e *actor.Engine) error {
	// Check if the Remote is already started.
	if r.state.Load() != stateInitialized {
		return fmt.Errorf("remote already started")
	}
	// Set the state of the Remote to stateRunning.
	r.state.Store(stateRunning)
	// Set the engine field to the provided Engine.
	r.engine = e
	var ln net.Listener
	var err error
	// Listen on the address of the Remote.
	switch r.config.TlsConfig {
	case nil:
		ln, err = net.Listen("tcp", r.addr)
	default:
		slog.Debug("remote using TLS for listening")
		ln, err = tls.Listen("tcp", r.addr, r.config.TlsConfig)
	}
	if err != nil {
		return fmt.Errorf("remote failed to listen: %w", err)
	}
	slog.Debug("listening", "addr", r.addr)
	// Create a new DRPC mux.
	mux := drpcmux.New()
	// Register the Remote with the DRPC server.
	err = DRPCRegisterRemote(mux, r.streamReader)
	if err != nil {
		return fmt.Errorf("failed to register remote: %w", err)
	}
	// Create a new DRPC server.
	s := drpcserver.New(mux)

	// Spawn a new streamRouter.
	r.streamRouterPID = r.engine.Spawn(newStreamRouter(r.engine, r.config.TlsConfig), "router", actor.WithInboxSize(1024*1024))
	slog.Debug("server started", "listenAddr", r.addr)
	// Initialize the stopWg field with a new WaitGroup and add 1 to it.
	r.stopWg = &sync.WaitGroup{}
	r.stopWg.Add(1)
	// Initialize the stopCh field with a new channel.
	r.stopCh = make(chan struct{})
	// Create a new context with a cancel function.
	ctx, cancel := context.WithCancel(context.Background())
	// Start a goroutine that serves the DRPC server until the stopCh channel is closed.
	go func() {
		defer r.stopWg.Done()
		err := s.Serve(ctx, ln)
		if err != nil {
			slog.Error("drpcserver", "err", err)
		} else {
			slog.Debug("drpcserver stopped")
		}
	}()
	// Start a goroutine that waits for the stopCh channel to be closed and then cancels the context.
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
	slog.Debug("stopping remote")
	r.state.Store(stateStopped)
	r.stopCh <- struct{}{}
	slog.Debug("stop signal sent")
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
