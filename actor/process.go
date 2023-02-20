package actor

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/anthdm/hollywood/log"
)

const restartDelay = time.Millisecond * 500

type Envelope struct {
	Msg    any
	Sender *PID
}

type Processer interface {
	Start()
	PID() *PID
	Send(*PID, any, *PID)
	Invoke([]Envelope)
	Shutdown()
}

type process struct {
	Opts

	inbox    Inboxer
	context  *Context
	pid      *PID
	restarts int32
}

func newProcess(e *Engine, opts Opts) *process {
	pid := NewPID(e.address, opts.Name, opts.Tags...)
	ctx := newContext(e, pid)

	return &process{
		pid:     pid,
		inbox:   NewInbox(opts.InboxSize),
		Opts:    opts,
		context: ctx,
	}
}

func applyMiddleware(rcv ReceiveFunc, middleware ...MiddlewareFunc) ReceiveFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		rcv = middleware[i](rcv)
	}
	return rcv
}

func (p *process) Invoke(msgs []Envelope) {
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		if _, ok := msg.Msg.(poisonPill); ok {
			p.cleanup()
			return
		}
		p.context.message = msg.Msg
		p.context.sender = msg.Sender
		recv := p.context.receiver
		if len(p.Opts.Middleware) > 0 {
			applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
		} else {
			recv.Receive(p.context)
		}
	}
}

func (p *process) Start() {
	defer func() {
		if p.MaxRestarts > 0 {
			if v := recover(); v != nil {
				p.tryRestart(v)
			}
		}
	}()

	recv := p.Producer()
	p.context.receiver = recv
	p.context.message = Initialized{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)

	p.inbox.Start(p)

	p.context.message = Started{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
	p.context.engine.EventStream.Publish(&ActivationEvent{PID: p.pid})

	log.Tracew("[PROCESS] started", log.M{
		"pid": p.pid,
	})
}

func (p *process) tryRestart(v any) {
	p.inbox.Close()
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
		p.Start()
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
	p.Start()
}

func (p *process) cleanup() {
	p.inbox.Close()
	p.context.engine.registry.remove(p.pid)
	p.context.message = Stopped{}
	applyMiddleware(p.context.receiver.Receive, p.Opts.Middleware...)(p.context)

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
	log.Tracew("[PROCESS] shutdown", log.M{
		"pid": p.pid,
	})
	// Send TerminationEvent to the eventstream
	p.context.engine.EventStream.Publish(&TerminationEvent{PID: p.pid})
}

func (p *process) PID() *PID { return p.pid }
func (p *process) Send(_ *PID, msg any, sender *PID) {
	p.inbox.Send(Envelope{Msg: msg, Sender: sender})
}
func (p *process) Shutdown() { p.cleanup() }
