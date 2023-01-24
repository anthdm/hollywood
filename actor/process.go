package actor

import (
	"fmt"
	"runtime/debug"

	"github.com/anthdm/hollywood/log"
)

type process struct {
	ProducerConfig

	inbox    chan any
	context  *Context
	pid      *PID
	restarts int
	quitch   chan struct{}
}

func NewProcess(e *Engine, cfg ProducerConfig) *process {
	pid := NewPID(e.address, cfg.Name)
	ctx := &Context{
		engine: e,
		pid:    pid,
	}

	return &process{
		pid:            pid,
		inbox:          make(chan any, 1000),
		ProducerConfig: cfg,
		context:        ctx,
		quitch:         make(chan struct{}, 1),
	}
}

func (p *process) start() *PID {
	recv := p.Producer()
	p.inbox <- Started{}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				p.restarts++
				log.Errorw("[ACTOR] restarting", log.M{
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
				p.context.message = msg
				recv.Receive(p.context)
			case <-p.quitch:
				close(p.inbox)
				break loop
			}
		}
		log.Tracew("[PROCESS] mailbox shutdown", log.M{
			"pid": p.pid,
		})
	}()

	return p.pid
}
