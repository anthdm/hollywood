package actor

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/anthdm/hollywood/log"
)

type Envelope struct {
	Msg    any
	Sender *PID
}

// Processer is an interface the abstracts the way a process behaves.
type Processer interface {
	Start()
	PID() *PID
	Send(*PID, any, *PID)
	Invoke([]Envelope)
	Shutdown(*sync.WaitGroup)
}

type process struct {
	Opts

	inbox    Inboxer
	context  *Context
	pid      *PID
	restarts int32

	mbuffer []Envelope
}

func newProcess(e *Engine, opts Opts) *process {
	pid := NewPID(e.address, opts.Name, opts.Tags...)
	ctx := newContext(e, pid)
	p := &process{
		pid:     pid,
		inbox:   NewInbox(opts.InboxSize),
		Opts:    opts,
		context: ctx,
		mbuffer: nil,
	}
	p.inbox.Start(p)
	return p
}

func applyMiddleware(rcv ReceiveFunc, middleware ...MiddlewareFunc) ReceiveFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		rcv = middleware[i](rcv)
	}
	return rcv
}

func (p *process) Invoke(msgs []Envelope) {
	var (
		// numbers of msgs that need to be processed.
		nmsg = len(msgs)
		// numbers of msgs that are processed.
		nproc = 0
	)
	defer func() {
		// If we recovered, we buffer up all the messages that we could not process
		// so we can retry them on the next restart.
		if v := recover(); v != nil {
			p.context.message = Stopped{}
			p.context.receiver.Receive(p.context)

			p.mbuffer = make([]Envelope, nmsg-nproc)
			for i := 0; i < nmsg-nproc; i++ {
				p.mbuffer[i] = msgs[i+nproc]
			}
			if p.Opts.MaxRestarts > 0 {
				p.tryRestart(v)
			}
		}
	}()
	for i := 0; i < len(msgs); i++ {
		nproc++
		msg := msgs[i]
		if pill, ok := msg.Msg.(poisonPill); ok {
			p.cleanup(pill.wg)
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
	recv := p.Producer()
	p.context.receiver = recv
	p.context.message = Initialized{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)

	p.context.message = Started{}
	applyMiddleware(recv.Receive, p.Opts.Middleware...)(p.context)
	p.context.engine.EventStream.Publish(&ActivationEvent{PID: p.pid})

	log.Tracew("[PROCESS] started", log.M{
		"pid": p.pid,
	})

	// If we have messages in our buffer, invoke them.
	if len(p.mbuffer) > 0 {
		p.Invoke(p.mbuffer)
		p.mbuffer = nil
	}
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
		time.Sleep(p.Opts.RestartDelay)
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
		p.cleanup(nil)
		return
	}
	// Restart the process after its restartDelay
	log.Errorw("[PROCESS] actor restarting", log.M{
		"n":           p.restarts,
		"maxRestarts": p.MaxRestarts,
		"pid":         p.pid,
		"reason":      v,
	})
	time.Sleep(p.Opts.RestartDelay)
	p.Start()
}

func (p *process) cleanup(wg *sync.WaitGroup) {
	p.inbox.Stop()
	p.context.engine.Registry.Remove(p.pid)
	p.context.message = Stopped{}
	applyMiddleware(p.context.receiver.Receive, p.Opts.Middleware...)(p.context)

	// We are a child if the parent context is not nil
	// No need for a mutex here, cause this is getting called inside the
	// the parents children foreach loop, which already locks.
	if p.context.parentCtx != nil {
		p.context.parentCtx.children.Delete(p.Name)
	}

	// We are a parent if we have children running, shutdown all the children.
	if p.context.children.Len() > 0 {
		children := p.context.Children()
		for _, pid := range children {
			if wg != nil {
				wg.Add(1)
			}
			proc := p.context.engine.Registry.get(pid)
			proc.Shutdown(wg)
		}
	}
	log.Tracew("[PROCESS] shutdown", log.M{
		"pid": p.pid,
	})
	// Send TerminationEvent to the eventstream
	p.context.engine.EventStream.Publish(&TerminationEvent{PID: p.pid})
	if wg != nil {
		wg.Done()
	}
}

func (p *process) PID() *PID { return p.pid }
func (p *process) Send(_ *PID, msg any, sender *PID) {
	p.inbox.Send(Envelope{Msg: msg, Sender: sender})
}
func (p *process) Shutdown(wg *sync.WaitGroup) { p.cleanup(wg) }
