package actor

import (
	"time"

	"github.com/anthdm/hollywood/log"
)

type ProducerConfig struct {
	Producer    Producer
	Name        string
	Tags        []string
	MaxRestarts int
	InboxSize   int
}

func DefaultProducerConfig(p Producer) ProducerConfig {
	return ProducerConfig{
		Producer:    p,
		MaxRestarts: 3,
		InboxSize:   100,
	}
}

type Receiver interface {
	Receive(*Context)
}

type Producer func() Receiver

type Engine struct {
	EventStream *EventStream

	address    string
	registry   *registry
	remote     Remoter
	deadLetter processer
}

type Remoter interface {
	Address() string
	Send(*PID, any)
	Start()
}

func NewEngine() *Engine {
	e := &Engine{
		EventStream: NewEventStream(),
		address:     "local",
	}
	e.registry = newRegistry(e)
	e.deadLetter = newDeadLetter()
	e.registry.add(e.deadLetter)
	return e
}

func (e *Engine) WithRemote(r Remoter) {
	e.remote = r
	e.address = r.Address()
	r.Start()
}

func (e *Engine) SpawnConfig(cfg ProducerConfig) *PID {
	return e.spawn(cfg).PID()
}

// Address returns the address of the actor engine. When there is
// no remote configured, the "local" address will be used, otherwise
// the listen address of the remote.
func (e *Engine) Address() string {
	return e.address
}

func (e *Engine) Spawn(p Producer, name string, tags ...string) *PID {
	pconf := DefaultProducerConfig(p)
	pconf.Name = name
	pconf.Tags = tags
	return e.spawn(pconf).PID()
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

func (e Engine) Poison(pid *PID) {
	proc := e.registry.get(pid)
	if proc != nil {
		e.sendLocal(pid, Stopped{})
		proc.Shutdown()
		e.registry.remove(pid)
	}
}

func (e *Engine) spawn(cfg ProducerConfig) processer {
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
