package actor

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/anthdm/hollywood/log"
)

const restartDelay = time.Millisecond * 500

type envelope struct {
	msg    any
	sender *PID
}

type processer interface {
	PID() *PID
	Send(*PID, any, *PID)
	Shutdown()
}

type process struct {
	Opts

	inbox    chan envelope
	context  *Context
	pid      *PID
	restarts int32
	quitch   chan struct{}
}

func newProcess(e *Engine, opts Opts) *process {
	pid := NewPID(e.address, opts.Name, opts.Tags...)
	ctx := newContext(e, pid)
	p := &process{
		pid:     pid,
		inbox:   make(chan envelope, opts.InboxSize),
		Opts:    opts,
		context: ctx,
		quitch:  make(chan struct{}, 1),
	}
	p.start()
	return p
}

func applyMiddleware(rcv ReceiveFunc, middleware ...MiddlewareFunc) ReceiveFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		rcv = middleware[i](rcv)
	}
	return rcv
}

func (p *process) start() *PID {
	recv := p.Producer()

	p.context.engine.EventStream.Publish(&ActivationEvent{PID: p.pid})
	p.context.receiver = recv
	p.context.message = Initialized{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)

	go func() {
		defer func() {
			if p.MaxRestarts > 0 {
				if v := recover(); v != nil {
					p.tryRestart(v)
				}
			}
		}()
		p.context.message = Started{}
		applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
		log.Debugw("[PROCESS] started", log.M{
			"pid": p.pid,
		})
	loop:
		for {
			select {
			case env := <-p.inbox:
				p.context.sender = env.sender
				if _, ok := env.msg.(poisonPill); ok {
					close(p.inbox)
					p.context.message = Stopped{}
					applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
					break loop
				}
				p.context.message = env.msg
				applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
			case <-p.quitch:
				close(p.inbox)
				break loop
			}
		}
		p.cleanup()
	}()

	return p.pid
}

func (p *process) tryRestart(v any) {
	p.restarts++
	// InternalError does not take the maximum restarts into account.
	// For now, InternalError is getting triggered when we are dialing
	// a remote node. By doing this, we can keep dialing until it comes
	// back up. NOTE: not sure if that is the best option. What if that
	// node never comes back up again?
	if msg, ok := v.(*InternalError); ok {
		log.Errorw(msg.From, log.M{
			"error": msg.Err,
		})
		time.Sleep(restartDelay)
		p.start()
		return
	}

	fmt.Println(string(debug.Stack()))
	// If we reach the max restarts, we shutdown the inbox and clean
	// everything up.
	if p.restarts == p.MaxRestarts {
		log.Errorw("[PROCESS] max restarts exceeded, shutting down...", log.M{
			"pid":      p.pid,
			"restarts": p.restarts,
		})
		close(p.quitch)
		return
	}
	// Restart the process after its restartDelay
	log.Errorw("[PROCESS] actor restarting", log.M{
		"n":           p.restarts,
		"maxRestarts": p.MaxRestarts,
		"pid":         p.pid,
		"reason":      v,
	})
	time.Sleep(restartDelay)
	p.start()
}

func (p *process) cleanup() {
	p.context.engine.registry.remove(p.pid)
	// We are a child if the parent context is not nil
	if p.context.parentCtx != nil {
		p.context.parentCtx.children.Delete(p.Name)
	}
	// We are a parent if we have children running
	if p.context.children.Len() > 0 {
		p.context.children.ForEach(func(name string, pid *PID) {
			p.context.engine.Poison(pid)
			log.Tracew("[PROCESS] shutting down child", log.M{
				"pid":   p.pid,
				"child": pid,
			})
		})
	}
	log.Tracew("[PROCESS] inbox shutdown", log.M{
		"pid": p.pid,
	})
	// Send TerminationEvent to the eventstream
	p.context.engine.EventStream.Publish(&TerminationEvent{PID: p.pid})
}

func (p *process) PID() *PID { return p.pid }
func (p *process) Send(_ *PID, msg any, sender *PID) {
	p.inbox <- envelope{
		msg:    msg,
		sender: sender,
	}
}
func (p *process) Shutdown() { close(p.quitch) }
