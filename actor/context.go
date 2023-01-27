package actor

import "github.com/anthdm/hollywood/log"

type Context struct {
	pid      *PID
	sender   *PID
	engine   *Engine
	message  any
	children map[string]*PID
}

func newContext(e *Engine, pid *PID) *Context {
	return &Context{
		engine:   e,
		pid:      pid,
		children: make(map[string]*PID),
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

func (c *Context) SpawnChild(p Producer, name string, tags ...string) *PID {
	pid := c.engine.Spawn(p, name, tags...)
	c.children[name] = pid
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
