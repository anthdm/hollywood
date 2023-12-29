package actor

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Envelope is a struct that encapsulates a message and its sender.
// It's used to handle message passing in the system.
type Envelope struct {
	Msg    any  // Msg holds the message content, can be of any type.
	Sender *PID // Sender is a pointer to the PID (process identifier) of the message sender.
}

// Processer is an interface that abstracts the behavior of a process.
// It defines a set of methods that any process should implement.
type Processer interface {
	Start()                   // Start initiates the process.
	PID() *PID                // PID returns a pointer to the process identifier.
	Send(*PID, any, *PID)     // Send handles sending a message to a process.
	Invoke([]Envelope)        // Invoke processes a batch of Envelopes.
	Shutdown(*sync.WaitGroup) // Shutdown handles the process shutdown, coordinating with the provided WaitGroup.
}

// process is a struct that implements the Processer interface.
// It represents the internal state and behavior of a process.
type process struct {
	Opts // Embeds Opts struct, inheriting its fields.

	inbox    Inboxer  // inbox is an interface to handle incoming messages.
	context  *Context // context stores the context of the process, used for various operations within the process.
	pid      *PID     // pid is a pointer to the process identifier.
	restarts int32    // restarts tracks the number of times the process has been restarted.

	mbuffer []Envelope // mbuffer is a slice that stores Envelopes, used for message buffering.
}

// newProcess creates a new instance of a process.
// It initializes the process with the provided Engine and options.
func newProcess(e *Engine, opts Opts) *process {
	// pid creates a new process identifier using Engine's address, the kind of the process, and its ID.
	pid := NewPID(e.address, opts.Kind+pidSeparator+opts.ID)

	ctx := newContext(e, pid)

	// p is a pointer to the new process being created with the specified pid, inbox, options, context, and message buffer.
	p := &process{
		pid:     pid,
		inbox:   NewInbox(opts.InboxSize), // Initializes the inbox with the specified size from the options.
		Opts:    opts,                     // Sets the process options.
		context: ctx,                      // Sets the process context.
		mbuffer: nil,                      // Initializes the message buffer as nil.
	}

	// Start the inbox to begin processing messages.
	p.inbox.Start(p)

	return p
}

// applyMiddleware is a function that applies a series of middleware functions to a ReceiveFunc.
// Middleware functions can modify or extend the behavior of the ReceiveFunc.
func applyMiddleware(rcv ReceiveFunc, middleware ...MiddlewareFunc) ReceiveFunc {

	for i := len(middleware) - 1; i >= 0; i-- {
		// Apply the middleware function to the ReceiveFunc.
		rcv = middleware[i](rcv)
	}

	// Return the modified ReceiveFunc after all middleware have been applied.
	return rcv
}

func (p *process) Invoke(msgs []Envelope) {
	var (
		// numbers of msgs that need to be processed.
		nmsg = len(msgs)
		// numbers of msgs that are processed.
		nproc = 0
		// FIXME: We could use nrpoc here, but for some reason placing nproc++ on the
		// bottom of the function it freezes some tests. Hence, I created a new counter
		// for bookkeeping.
		processed = 0
	)
	defer func() {
		// If we recovered, we buffer up all the messages that we could not process
		// so we can retry them on the next restart.
		if v := recover(); v != nil {
			p.context.message = Stopped{}
			p.context.receiver.Receive(p.context)

			p.mbuffer = make([]Envelope, nmsg-nproc)
			for i := 0; i < nmsg-nproc; i++ {
				p.mbuffer[i] = msgs[i+nproc]
			}
			p.tryRestart(v)
		}
	}()
	for i := 0; i < len(msgs); i++ {
		nproc++
		msg := msgs[i]
		if pill, ok := msg.Msg.(poisonPill); ok {
			// If we need to gracefuly stop, we process all the messages
			// from the inbox, otherwise we ignore and cleanup.
			if pill.graceful {
				msgsToProcess := msgs[processed:]
				for _, m := range msgsToProcess {
					p.invokeMsg(m)
				}
			}
			p.cleanup(pill.wg)
			return
		}
		p.invokeMsg(msg)
		processed++
	}
}

func (p *process) invokeMsg(msg Envelope) {
	// suppress poison pill messages here. they're private to the actor engine.
	if _, ok := msg.Msg.(poisonPill); ok {
		return
	}
	p.context.message = msg.Msg
	p.context.sender = msg.Sender
	recv := p.context.receiver
	if len(p.Opts.Middleware) > 0 {
		applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
	} else {
		recv.Receive(p.context)
	}
}

// Start is a method of the process type. It initializes and starts the process's execution.
func (p *process) Start() {
	// recv holds the ReceiveFunc produced by the process's Producer.
	recv := p.Producer()
	p.context.receiver = recv

	// defer a function to handle any panic that may occur in the process.
	defer func() {
		// recover from panic if any occurs.
		if v := recover(); v != nil {
			// Set the process's message to Stopped and invoke the receiver's Receive method.
			p.context.message = Stopped{}
			p.context.receiver.Receive(p.context)

			// Attempt to restart the process based on the panic value.
			p.tryRestart(v)
		}
	}()

	// Initialize the process by setting its message to Initialized and applying middleware.
	p.context.message = Initialized{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)

	// Set the process's message to Started, apply middleware, and start the receiver.
	p.context.message = Started{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)

	// Broadcast an ActorStartedEvent to the engine, marking the start of the process.
	p.context.engine.BroadcastEvent(ActorStartedEvent{PID: p.pid, Timestamp: time.Now()})

	// If there are messages in the process's buffer, invoke them.
	if len(p.mbuffer) > 0 {
		p.Invoke(p.mbuffer)
		p.mbuffer = nil // Clear the message buffer after invocation.
	}
}

func (p *process) tryRestart(v any) {
	// InternalError does not take the maximum restarts into account.
	// For now, InternalError is getting triggered when we are dialing
	// a remote node. By doing this, we can keep dialing until it comes
	// back up. NOTE: not sure if that is the best option. What if that
	// node never comes back up again?
	if msg, ok := v.(*InternalError); ok {
		slog.Error(msg.From, "err", msg.Err)
		time.Sleep(p.Opts.RestartDelay)
		p.Start()
		return
	}
	stackTrace := debug.Stack()
	fmt.Println(string(stackTrace))
	// If we reach the max restarts, we shutdown the inbox and clean
	// everything up.
	if p.restarts == p.MaxRestarts {
		p.context.engine.BroadcastEvent(ActorMaxRestartsExceededEvent{
			PID:       p.pid,
			Timestamp: time.Now(),
		})
		p.cleanup(nil)
		return
	}

	p.restarts++
	// Restart the process after its restartDelay
	p.context.engine.BroadcastEvent(ActorRestartedEvent{
		PID:        p.pid,
		Timestamp:  time.Now(),
		Stacktrace: stackTrace,
		Reason:     v,
		Restarts:   p.restarts,
	})
	time.Sleep(p.Opts.RestartDelay)
	p.Start()
}

func (p *process) cleanup(wg *sync.WaitGroup) {
	p.inbox.Stop()
	p.context.engine.Registry.Remove(p.pid)
	p.context.message = Stopped{}
	applyMiddleware(p.context.receiver.Receive, p.Opts.Middleware...)(p.context)

	// We are a child if the parent context is not nil
	// No need for a mutex here, cause this is getting called inside the
	// the parents children foreach loop, which already locks.
	if p.context.parentCtx != nil {
		p.context.parentCtx.children.Delete(p.Kind)
	}

	// We are a parent if we have children running, shutdown all the children.
	if p.context.children.Len() > 0 {
		children := p.context.Children()
		for _, pid := range children {
			if wg != nil {
				wg.Add(1)
			}
			proc := p.context.engine.Registry.get(pid)
			proc.Shutdown(wg)
		}
	}
	p.context.engine.BroadcastEvent(ActorStoppedEvent{PID: p.pid, Timestamp: time.Now()})
	if wg != nil {
		wg.Done()
	}
}

// PID is a method of the process type. It returns the process identifier.
func (p *process) PID() *PID {
	return p.pid
}

// Send is a method of the process type for sending a message.
// It places an Envelope containing the message and sender into the process's inbox.
func (p *process) Send(_ *PID, msg any, sender *PID) {
	// Send an Envelope with the message and sender to the process's inbox.
	p.inbox.Send(Envelope{Msg: msg, Sender: sender})
}

// Shutdown is a method of the process type that initiates the process's shutdown procedure.
// It takes a WaitGroup to coordinate the shutdown with other goroutines.
func (p *process) Shutdown(wg *sync.WaitGroup) {
	p.cleanup(wg)
}
