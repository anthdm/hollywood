package actor

import (
	"os"
	"sync"
	"time"

	"github.com/anthdm/hollywood/log"
)

type ProducerConfig struct {
	Producer    Producer
	Name        string
	MaxRestarts int
}

func DefaultProducerConfig(p Producer) ProducerConfig {
	return ProducerConfig{
		Producer:    p,
		MaxRestarts: 3,
	}
}

type Receiver interface {
	Receive(*Context)
}

type Producer func() Receiver

type Registry struct {
	mu    sync.RWMutex
	procs map[*PID]*Process
}

func NewRegistry() *Registry {
	return &Registry{
		procs: make(map[*PID]*Process),
	}
}

func (r *Registry) remove(pid *PID) {
	r.mu.Lock()
	delete(r.procs, pid)
	r.mu.Unlock()
}

func (r *Registry) add(pid *PID, proc *Process) {
	r.mu.Lock()
	r.procs[pid] = proc
	r.mu.Unlock()
}

func (r *Registry) get(pid *PID) *Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.procs[pid]
}

type Engine struct {
	EventStream *EventStream

	registry *Registry
}

func NewEngine() *Engine {
	return &Engine{
		registry:    NewRegistry(),
		EventStream: NewEventStream(),
	}
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

	return pid
}

func (e *Engine) Send(pid *PID, msg any) {
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
	}
}

type Process struct {
	ProducerConfig

	mailbox  chan any
	context  *Context
	pid      *PID
	restarts int
	quitch   chan struct{}
}

func NewProcess(e *Engine, cfg ProducerConfig) *Process {
	host, _ := os.Hostname()
	pid := NewPID(host, cfg.Name)
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
	}
}

func (p *Process) start() *PID {
	recv := p.Producer()
	p.mailbox <- Started{}

	go func() {
		<-p.quitch
		p.mailbox <- Stopped{}
		close(p.mailbox)
	}()

	start := time.Now()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				p.restarts++
				if p.restarts == p.MaxRestarts {
					log.Errorw("[ACTOR] restart", log.M{"n": p.restarts})
					return
				}
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
