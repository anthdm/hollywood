package actor

import (
	"context"
	"fmt"
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

	address  string
	registry *registry
	remote   Remoter
}

type Remoter interface {
	Address() string
	Send(*PID, any)
}

func NewEngine() *Engine {
	e := &Engine{
		registry:    newRegistry(),
		EventStream: NewEventStream(),
		address:     "local",
	}

	return e
}

func (e *Engine) WithRemote(r Remoter) {
	e.remote = r
	e.address = r.Address()
}

func (e *Engine) SpawnConfig(cfg ProducerConfig) *PID {
	return e.spawn(cfg)
}

func (e *Engine) Spawn(p Producer, name string, tags ...string) *PID {
	pconf := DefaultProducerConfig(p)
	pconf.Name = name
	pconf.Tags = tags
	return e.spawn(pconf)
}

func (e *Engine) spawn(cfg ProducerConfig) *PID {
	proc := newProcess(e, cfg)
	proc.start()
	e.registry.add(proc)

	return proc.pid
}

func (e *Engine) Address() string {
	return e.address
}

func (e *Engine) Request(pid *PID, msg any, timeout time.Duration) (any, error) {
	proc := e.registry.get(pid)
	if proc == nil {
		return nil, fmt.Errorf("pid [%s] not found in registry", pid)
	}
	proc.context.respch = make(chan any, 1)

	e.Send(pid, msg)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-proc.context.respch:
		close(proc.context.respch)
		return res, nil
	}
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

func (e *Engine) sendLocal(pid *PID, msg any) {
	proc := e.registry.get(pid)
	if proc != nil {
		proc.inbox <- msg
		return
	}
	dl := &DeadLetter{
		PID:     pid,
		Message: msg,
		Sender:  nil,
	}

	log.Warnw("DEADLETTER", log.M{"pid": dl.PID, "sender": dl.Sender})
}

func (e Engine) Poison(pid *PID) {
	proc := e.registry.get(pid)
	if proc != nil {
		e.sendLocal(pid, Stopped{})
		close(proc.quitch)
		e.registry.remove(pid)
	}
}

func (e *Engine) isLocalMessage(pid *PID) bool {
	return e.address == pid.Address
}
