package actor

import (
	"context"
	"math/rand"
	"strconv"
	"time"
)

type Response struct {
	engine  *Engine
	pid     *PID
	result  chan any
	timeout time.Duration
}

func NewResponse(e *Engine, timeout time.Duration) *Response {
	return &Response{
		engine:  e,
		result:  make(chan any, 1),
		timeout: timeout,
		pid:     NewPID(e.address, "response", strconv.Itoa(rand.Intn(100000))),
	}
}

func (r *Response) Result() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer func() {
		cancel()
		r.engine.registry.remove(r.pid)
	}()

	select {
	case resp := <-r.result:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Response) Send(_ *PID, msg any) {
	r.result <- msg
}

func (r *Response) PID() *PID { return r.pid }
func (r *Response) Shutdown() {}
