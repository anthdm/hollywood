package actor

import "github.com/anthdm/hollywood/log"

type WithSender struct {
	Message any
	Sender  *PID
}

type Context struct {
	pid     *PID
	sender  *PID
	engine  *Engine
	message any
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
