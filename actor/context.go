package actor

type Context struct {
	pid     *PID
	engine  *Engine
	message any
	respch  chan any
}

func (c *Context) Respond(msg any) {
	c.respch <- msg
}

func (c *Context) PID() *PID {
	return c.pid
}

func (c *Context) Engine() *Engine {
	return c.engine
}

func (c *Context) Message() any {
	return c.message
}
