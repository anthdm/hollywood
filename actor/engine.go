package actor

import (
	"time"

	"github.com/anthdm/hollywood/log"
)

type Remoter interface {
	Address() string
	Send(*PID, any, *PID)
	Start()
}

// Producer is any function that can return a Receiver
type Producer func() Receiver

// Receiver is an interface that can receive and process messages.
type Receiver interface {
	Receive(*Context)
}

// Engine represents the actor engine.
type Engine struct {
	EventStream *EventStream

	address    string
	registry   *registry
	remote     Remoter
	deadLetter processer
}

// NewEngine returns a new actor Engine.
func NewEngine() *Engine {
	e := &Engine{
		EventStream: NewEventStream(),
		address:     "local",
	}
	e.registry = newRegistry(e)
	e.deadLetter = newDeadLetter(e.EventStream)
	e.registry.add(e.deadLetter)
	return e
}

// WithRemote returns a new actor Engine with the given Remoter,
// and will call its Start function
func (e *Engine) WithRemote(r Remoter) {
	e.remote = r
	e.address = r.Address()
	r.Start()
}

// Spawn spawns a process that will producer by the given Producer and
// can be configured with the given opts.
func (e *Engine) Spawn(p Producer, name string, opts ...OptFunc) *PID {
	options := DefaultOpts(p)
	options.Name = name
	for _, opt := range opts {
		opt(&options)
	}
	return e.spawn(options).PID()
}

func (e *Engine) SpawnFunc(f func(*Context), id string, opts ...OptFunc) *PID {
	return e.Spawn(newFuncReceiver(f), id, opts...)
}

// Address returns the address of the actor engine. When there is
// no remote configured, the "local" address will be used, otherwise
// the listen address of the remote.
func (e *Engine) Address() string {
	return e.address
}

// Request sends the given message to the given PID as a "Request", returning
// a response that will resolve in the future. Calling Response.Result() will
// block until the deadline is exceeded or the response is being resolved.
func (e *Engine) Request(pid *PID, msg any, timeout time.Duration) *Response {
	resp := NewResponse(e, timeout)
	e.registry.add(resp)

	e.SendWithSender(pid, msg, resp.PID())

	return resp
}

// SendWithSender will send the given message to the given PID with the
// given sender. Receivers receiving this message can check the sender
// by calling Context.Sender().
func (e *Engine) SendWithSender(pid *PID, msg any, sender *PID) {
	e.send(pid, msg, sender)
}

// Send sends the given message to the given PID. If the message cannot be
// delivered due to the fact that the given process is not registered.
// The message will be send to the DeadLetter process instead.
func (e *Engine) Send(pid *PID, msg any) {
	e.send(pid, msg, nil)
}

func (e *Engine) send(pid *PID, msg any, sender *PID) {
	if e.isLocalMessage(pid) {
		e.sendLocal(pid, msg, sender)
		return
	}
	if e.remote == nil {
		log.Errorw("[ENGINE] failed sending messsage", log.M{
			"err": "engine has no remote configured",
		})
		return
	}
	e.remote.Send(pid, msg, sender)
}

// Poison will send a poisonPill to the process that is associated with the given PID.
// The process will shut down once it processed all its messages before the poisonPill
// was received.
func (e *Engine) Poison(pid *PID) {
	proc := e.registry.get(pid)
	if proc != nil {
		e.sendLocal(pid, poisonPill{}, nil)
	}
}

func (e *Engine) spawn(cfg Opts) processer {
	return newProcess(e, cfg)
}

func (e *Engine) sendLocal(pid *PID, msg any, sender *PID) {
	proc := e.registry.get(pid)
	if proc != nil {
		proc.Send(pid, msg, sender)
	}
}

func (e *Engine) isLocalMessage(pid *PID) bool {
	return e.address == pid.Address
}

type funcReceiver struct {
	f func(*Context)
}

func newFuncReceiver(f func(*Context)) Producer {
	return func() Receiver {
		return &funcReceiver{
			f: f,
		}
	}
}

func (r *funcReceiver) Receive(c *Context) {
	r.f(c)
}
