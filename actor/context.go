package actor

import (
	"github.com/anthdm/hollywood/log"
	"github.com/anthdm/hollywood/safemap"
)

type Context struct {
	pid     *PID
	sender  *PID
	engine  *Engine
	message any
	// the context of the parent, if this is the context of a child.
	// we need this so we can remove the child from the parent Context
	// when the child dies.
	parentCtx *Context
	children  *safemap.SafeMap[string, *PID]
}

func newContext(e *Engine, pid *PID) *Context {
	return &Context{
		engine:   e,
		pid:      pid,
		children: safemap.New[string, *PID](),
	}
}

func (c *Context) Respond(msg any) {
	if c.sender == nil {
		log.Warnw("[RESPOND] context got no sender", log.M{
			"pid": c.PID(),
		})
		return
	}
	c.engine.Send(c.sender, msg)
}

// SpawnChild will spawn the given Producer as a child of the current Context.
// If the parent process dies, all the children will be automatically shutdown gracefully.
// Hence, all children will receive the Stopped message.
func (c *Context) SpawnChild(p Producer, name string, tags ...string) *PID {
	cfg := Opts{
		Producer: p,
		Name:     name,
		Tags:     tags,
	}
	proc := c.engine.spawn(cfg)
	proc.(*process).context.parentCtx = c
	c.children.Set(name, proc.PID())
	return proc.PID()
}

// GetChild will return the PID of the child (if any) by the given name/id.
// PID will be nil if it could not find it.
func (c *Context) GetChild(id string) *PID {
	pid, _ := c.children.Get(id)
	return pid
}

func (c *Context) Send(pid *PID, msg any) {
	c.engine.Send(pid, msg)
}

func (c *Context) PID() *PID {
	return c.pid
}

func (c *Context) Sender() *PID {
	return c.sender
}

func (c *Context) Engine() *Engine {
	return c.engine
}

func (c *Context) Message() any {
	return c.message
}
