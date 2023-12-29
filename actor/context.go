package actor

import (
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/safemap"
)

// Context struct represents the execution context of a process in the actor system.
// It contains essential data required for a process's operation and interaction with other processes.
type Context struct {
	pid      *PID     // pid is the process identifier of the current context.
	sender   *PID     // sender is the PID of the process that sent the current message.
	engine   *Engine  // engine is a reference to the Engine that this context is part of.
	receiver Receiver // receiver is the message receiver associated with this context.
	message  any      // message is the current message being processed.
	// parentCtx is the context of the parent process (if this is a child process's context).
	// It is used to manage the lifecycle of child processes within a parent context.
	parentCtx *Context
	// children is a map of child process identifiers, allowing the context to keep track of its children.
	children *safemap.SafeMap[string, *PID]
}

// newContext creates and returns a new instance of Context for a given Engine and PID.
// It initializes a new context with a process identifier and an empty map for child processes.
func newContext(e *Engine, pid *PID) *Context {
	return &Context{
		engine:   e,                           // Set the Engine reference.
		pid:      pid,                         // Set the process identifier.
		children: safemap.New[string, *PID](), // Initialize an empty map for child PIDs.
	}
}

// Receiver returns the underlying receiver of this Context.
func (c *Context) Receiver() Receiver {
	return c.receiver
}

// See Engine.Request for information. This is just a helper function doing that
// calls Request on the underlying Engine. c.Engine().Request().
func (c *Context) Request(pid *PID, msg any, timeout time.Duration) *Response {
	return c.engine.Request(pid, msg, timeout)
}

// Respond will sent the given message to the sender of the current received message.
func (c *Context) Respond(msg any) {
	if c.sender == nil {
		slog.Warn("context got no sender", "func", "Respond", "pid", c.PID())
		return
	}
	c.engine.Send(c.sender, msg)
}

// SpawnChild will spawn the given Producer as a child of the current Context.
// If the parent process dies, all the children will be automatically shutdown gracefully.
// Hence, all children will receive the Stopped message.
func (c *Context) SpawnChild(p Producer, name string, opts ...OptFunc) *PID {
	options := DefaultOpts(p)
	options.Kind = c.PID().ID + pidSeparator + name
	for _, opt := range opts {
		opt(&options)
	}
	// Check if we got an ID, generate otherwise
	if len(options.ID) == 0 {
		id := strconv.Itoa(rand.Intn(math.MaxInt))
		options.ID = id
	}
	proc := newProcess(c.engine, options)
	proc.context.parentCtx = c
	pid := c.engine.SpawnProc(proc)
	c.children.Set(pid.ID, pid)

	return proc.PID()
}

// SpawnChildFunc spawns the given function as a child Receiver of the current
// Context.
func (c *Context) SpawnChildFunc(f func(*Context), name string, opts ...OptFunc) *PID {
	return c.SpawnChild(newFuncReceiver(f), name, opts...)
}

// Send will send the given message to the given PID.
// This will also set the sender of the message to
// the PID of the current Context. Hence, the receiver
// of the message can call Context.Sender() to know
// the PID of the process that sent this message.
func (c *Context) Send(pid *PID, msg any) {
	c.engine.SendWithSender(pid, msg, c.pid)
}

// SendRepeat will send the given message to the given PID each given interval.
// It will return a SendRepeater struct that can stop the repeating message by calling Stop().
func (c *Context) SendRepeat(pid *PID, msg any, interval time.Duration) SendRepeater {
	sr := SendRepeater{
		engine:   c.engine,
		self:     c.pid,
		target:   pid.CloneVT(),
		interval: interval,
		msg:      msg,
		cancelch: make(chan struct{}, 1),
	}
	sr.start()
	return sr
}

// Forward will forward the current received message to the given PID.
// This will also set the "forwarder" as the sender of the message.
func (c *Context) Forward(pid *PID) {
	c.engine.SendWithSender(pid, c.message, c.pid)
}

// GetPID returns the PID of the process found by the given id.
// Returns nil when it could not find any process..
func (c *Context) GetPID(id string) *PID {
	proc := c.engine.Registry.getByID(id)
	if proc != nil {
		return proc.PID()
	}
	return nil
}

// Parent returns the PID of the process that spawned the current process.
func (c *Context) Parent() *PID {
	if c.parentCtx != nil {
		return c.parentCtx.pid
	}
	return nil
}

// Child will return the PID of the child (if any) by the given name/id.
// PID will be nil if it could not find it.
func (c *Context) Child(id string) *PID {
	pid, _ := c.children.Get(id)
	return pid
}

// Children returns all child PIDs for the current process.
func (c *Context) Children() []*PID {
	pids := make([]*PID, c.children.Len())
	i := 0
	c.children.ForEach(func(_ string, child *PID) {
		pids[i] = child
		i++
	})
	return pids
}

// PID returns the PID of the process that belongs to the context.
func (c *Context) PID() *PID {
	return c.pid
}

// Sender, when available, returns the PID of the process that sent the
// current received message.
func (c *Context) Sender() *PID {
	return c.sender
}

// Engine returns a pointer to the underlying Engine.
func (c *Context) Engine() *Engine {
	return c.engine
}

// Message returns the message that is currently being received.
func (c *Context) Message() any {
	return c.message
}
