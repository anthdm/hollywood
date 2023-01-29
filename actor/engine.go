package actor

import (
	"time"

	"github.com/anthdm/hollywood/log"
)

type Opts struct {
	Producer    Producer
	Name        string
	Tags        []string
	MaxRestarts int32
	InboxSize   int
	WithHooks   bool
}

func DefaultOpts(p Producer) Opts {
	return Opts{
		Producer:    p,
		MaxRestarts: 3,
		InboxSize:   1000,
		WithHooks:   false,
	}
}

type Remoter interface {
	Address() string
	Send(*PID, any)
	Start()
}

// Producer is any function that can return a Receiver
type Producer func() Receiver

type Receiver interface {
	Receive(*Context)
}

type hookReceiver struct {
	r Receiver
}

type Hooker interface {
	OnInit(*Context)
	OnStart(*Context)
	OnStop(*Context)
}

func (h hookReceiver) Receive(ctx *Context) {
	switch ctx.Message().(type) {
	case Started:
		h.r.(Hooker).OnStart(ctx)
	case Stopped:
		h.r.(Hooker).OnStop(ctx)
	case Initialized:
		h.r.(Hooker).OnInit(ctx)
	}
	h.r.Receive(ctx)
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

func (e *Engine) Spawn(p Producer, name string, tags ...string) *PID {
	opts := DefaultOpts(p)
	opts.Name = name
	opts.Tags = tags
	return e.spawn(opts).PID()
}

func (e *Engine) SpawnOpts(cfg Opts) *PID {
	return e.spawn(cfg).PID()
}

func (e *Engine) SpawnFunc(f func(*Context), id string, tags ...string) *PID {
	return e.Spawn(newFuncReceiver(f), "foo", tags...)
}

// Address returns the address of the actor engine. When there is
// no remote configured, the "local" address will be used, otherwise
// the listen address of the remote.
func (e *Engine) Address() string {
	return e.address
}

func (e *Engine) Request(pid *PID, msg any, timeout time.Duration) *Response {
	resp := NewResponse(e, timeout)
	e.registry.add(resp)

	e.SendWithSender(pid, msg, resp.PID())

	return resp
}

func (e *Engine) SendWithSender(pid *PID, msg any, sender *PID) {
	m := &WithSender{
		Sender:  sender,
		Message: msg,
	}
	e.Send(pid, m)
}

func (e *Engine) Send(pid *PID, msg any) {
	if e.isLocalMessage(pid) {
		e.sendLocal(pid, msg)
		return
	}
	if e.remote == nil {
		log.Errorw("[ENGINE] failed sending messsage", log.M{
			"err": "engine has no remote configured",
		})
		return
	}
	e.remote.Send(pid, msg)
}

// Poison will shutdown the process that is associated with the given PID.
// If the process had any children, they will be shutdowned too.
func (e *Engine) Poison(pid *PID) {
	proc := e.registry.get(pid)
	if proc != nil {
		e.sendLocal(pid, Stopped{})
		proc.Shutdown()
		e.registry.remove(pid)
	}
}

func (e *Engine) spawn(cfg Opts) processer {
	proc := newProcess(e, cfg)
	e.registry.add(proc)
	return proc
}

func (e *Engine) sendLocal(pid *PID, msg any) {
	proc := e.registry.get(pid)
	if proc != nil {
		proc.Send(pid, msg)
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
