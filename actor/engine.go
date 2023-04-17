package actor

import (
	"sync"
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

// Config holds configuration for the actor Engine.
type Config struct {
	// PIDSeparator separates a process ID when printed out
	// in a string representation. The default separator is "/".
	// pid := NewPID("127.0.0.1:4000", "foo", "bar")
	// 127.0.0.1:4000/foo/bar
	PIDSeparator string
}

// Engine represents the actor engine.
type Engine struct {
	EventStream *EventStream
	Registry    *Registry

	address    string
	remote     Remoter
	deadLetter Processer
}

// NewEngine returns a new actor Engine.
func NewEngine(cfg ...Config) *Engine {
	e := &Engine{
		EventStream: NewEventStream(),
		address:     LocalLookupAddr,
	}
	if len(cfg) == 1 {
		e.configure(cfg[0])
	}
	e.Registry = newRegistry(e)
	e.deadLetter = newDeadLetter(e.EventStream)
	e.Registry.add(e.deadLetter)
	return e
}

func (e *Engine) configure(cfg Config) {
	if cfg.PIDSeparator != "" {
		pidSeparator = cfg.PIDSeparator
	}
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
	proc := newProcess(e, options)
	return e.SpawnProc(proc)
}

func (e *Engine) SpawnFunc(f func(*Context), id string, opts ...OptFunc) *PID {
	return e.Spawn(newFuncReceiver(f), id, opts...)
}

// SpawnProc spawns the give Processer. This function is usefull when working
// with custom created Processes. Take a look at the streamWriter as an example.
func (e *Engine) SpawnProc(p Processer) *PID {
	e.Registry.add(p)
	p.Start()
	return p.PID()
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
	e.Registry.add(resp)

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
		e.SendLocal(pid, msg, sender)
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

type SendRepeater struct {
	engine   *Engine
	self     *PID
	target   *PID
	msg      any
	interval time.Duration
	cancelch chan struct{}
}

func (sr SendRepeater) start() {
	ticker := time.NewTicker(sr.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				sr.engine.SendWithSender(sr.target, sr.msg, sr.self)
			case <-sr.cancelch:
				return
			}
		}
	}()
}

func (sr SendRepeater) Stop() {
	close(sr.cancelch)
}

// SendRepeat will send the given message to the given PID each given interval.
// It will return a SendRepeater struct that can stop the repeating message by calling Stop().
func (e *Engine) SendRepeat(pid *PID, msg any, interval time.Duration) SendRepeater {
	clonedPID := *pid.CloneVT()
	sr := SendRepeater{
		engine:   e,
		self:     nil,
		target:   &clonedPID,
		interval: interval,
		msg:      msg,
		cancelch: make(chan struct{}, 1),
	}
	sr.start()
	return sr
}

// Poison will send a poisonPill to the process that is associated with the given PID.
// The process will shut down once it processed all its messages before the poisonPill
// was received. If given a WaitGroup, you can wait till the process is completely shutdown.
func (e *Engine) Poison(pid *PID, wg ...*sync.WaitGroup) {
	var _wg *sync.WaitGroup
	if len(wg) > 0 {
		_wg = wg[0]
		_wg.Add(1)
	}
	proc := e.Registry.get(pid)
	if proc != nil {
		e.SendLocal(pid, poisonPill{_wg}, nil)
	}
}

func (e *Engine) SendLocal(pid *PID, msg any, sender *PID) {
	proc := e.Registry.get(pid)
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
