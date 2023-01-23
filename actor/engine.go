package actor

import (
	"os"
	"time"

	"github.com/anthdm/hollywood/log"
)

type ProducerConfig struct {
	Producer    Producer
	Name        string
	MaxRestarts int
	MailboxSize int
}

func DefaultProducerConfig(p Producer) ProducerConfig {
	return ProducerConfig{
		Producer:    p,
		MaxRestarts: 3,
		MailboxSize: 100,
	}
}

type Receiver interface {
	Receive(*Context)
}

type Producer func() Receiver

type Engine struct {
	EventStream *EventStream

	address  string
	registry *Registry
	remote   Remoter
}

type Remoter interface {
	Address() string
	Send(*PID, any)
}

func NewEngine() *Engine {
	var (
		host string
		err  error
	)
	host, err = os.Hostname()
	if err != nil {
		host = "local"
	}
	return &Engine{
		registry:    NewRegistry(),
		EventStream: NewEventStream(),
		address:     host,
	}
}

func (e *Engine) SetRemote(r Remoter) {
	e.remote = r
	e.address = r.Address()
}

func (e *Engine) SpawnConfig(cfg ProducerConfig) *PID {
	return e.spawn(cfg)
}

func (e *Engine) Spawn(p Producer, name string) *PID {
	pconf := DefaultProducerConfig(p)
	pconf.Name = name
	return e.spawn(pconf)
}

func (e *Engine) spawn(cfg ProducerConfig) *PID {
	proc := NewProcess(e, cfg)
	pid := proc.start()

	e.registry.add(pid, proc)

	log.Infow("[ACTOR] spawned", log.M{"pid": pid})

	return pid
}

func (e *Engine) Address() string {
	return e.address
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
		proc.mailbox <- msg
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
		close(proc.quitch)
		e.registry.remove(pid)
	}
}

type Process struct {
	ProducerConfig

	mailbox    chan any
	context    *Context
	pid        *PID
	restarts   int
	quitch     chan struct{}
	hardquitch chan struct{}
}

func NewProcess(e *Engine, cfg ProducerConfig) *Process {
	pid := NewPID(e.address, cfg.Name)
	ctx := &Context{
		engine: e,
		pid:    pid,
	}

	return &Process{
		pid:            pid,
		mailbox:        make(chan any, 1000),
		ProducerConfig: cfg,
		context:        ctx,
		quitch:         make(chan struct{}, 1),
		hardquitch:     make(chan struct{}, 1),
	}
}

func (p *Process) start() *PID {
	recv := p.Producer()
	p.mailbox <- Started{}

	go func() {
		select {
		case <-p.quitch:
			p.mailbox <- Stopped{}
			close(p.mailbox)
			break
		case <-p.hardquitch:
			break
		}
		log.Tracew("[ACTOR] process terminated", log.M{
			"pid": p.pid,
		})
	}()

	start := time.Now()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				p.restarts++
				if p.restarts == p.MaxRestarts {
					log.Errorw("[ACTOR] terminated", log.M{
						"pid":    p.pid,
						"reason": "max retries exceeded",
					})
					close(p.hardquitch)
					return
				}
				log.Errorw("[ACTOR] restart", log.M{
					"n":           p.restarts,
					"maxRestarts": p.MaxRestarts,
					"pid":         p.pid,
					"reason":      err,
				})
				p.start()
			}
			log.Debugw("[ACTOR] stopped", log.M{
				"pid":    p.pid,
				"active": time.Since(start),
			})
		}()

		for msg := range p.mailbox {
			p.context.message = msg
			recv.Receive(p.context)
		}
	}()

	return p.pid
}

func (e *Engine) isLocalMessage(pid *PID) bool {
	return e.address == pid.Address
}
