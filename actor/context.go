package actor

type Context struct {
	pid     *PID
	engine  *Engine
	message any
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
