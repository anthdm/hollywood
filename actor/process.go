package actor

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/anthdm/hollywood/log"
)

type processer interface {
	PID() *PID
	Send(*PID, any)
	Shutdown()
}

type process struct {
	ProducerConfig

	inbox    chan any
	context  *Context
	pid      *PID
	restarts int
	quitch   chan struct{}
}

func newProcess(e *Engine, cfg ProducerConfig) *process {
	pid := NewPID(e.address, cfg.Name, cfg.Tags...)
	ctx := &Context{
		engine: e,
		pid:    pid,
	}
	p := &process{
		pid:            pid,
		inbox:          make(chan any, 1000),
		ProducerConfig: cfg,
		context:        ctx,
		quitch:         make(chan struct{}, 1),
	}
	p.start()
	return p
}

type InternalError struct {
	From string
	Err  error
}

func (p *process) start() *PID {
	recv := p.Producer()
	p.inbox <- Started{}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				if msg, ok := err.(*InternalError); ok {
					time.Sleep(time.Second * 1)
					log.Errorw(msg.From, log.M{
						"error": msg.Err,
					})
					p.start()
					return
				}

				p.restarts++
				if p.restarts == p.MaxRestarts {
					log.Errorw("[PROCESS] max restarts exceeded, shutting down...", log.M{
						"pid": p.pid,
					})
					close(p.quitch)
				}

				log.Errorw("[PROCESS] actor restarting", log.M{
					"n":           p.restarts,
					"maxRestarts": p.MaxRestarts,
					"pid":         p.pid,
					"reason":      err,
				})
				fmt.Println(string(debug.Stack()))
				p.start()
			}
		}()

	loop:
		for {
			select {
			case msg := <-p.inbox:
				switch m := msg.(type) {
				case *WithSender:
					p.context.sender = m.Sender
					p.context.message = m.Message
				default:
					p.context.message = m
				}
				recv.Receive(p.context)
			case <-p.quitch:
				close(p.inbox)
				break loop
			}
		}
		p.context.engine.registry.remove(p.pid)
		log.Tracew("[PROCESS] inbox shutdown", log.M{
			"pid": p.pid,
		})
	}()

	return p.pid
}

func (p *process) PID() *PID            { return p.pid }
func (p *process) Send(_ *PID, msg any) { p.inbox <- msg }
func (p *process) Shutdown()            { close(p.quitch) }
