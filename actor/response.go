package actor

import (
	"context"
	"math"
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
		pid:     NewPID(e.address, "response"+pidSeparator+strconv.Itoa(rand.Intn(math.MaxInt32))),
	}
}

func (r *Response) Result() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer func() {
		cancel()
		r.engine.Registry.Remove(r.pid)
	}()

	select {
	case resp := <-r.result:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Response) Send(_ *PID, msg any, _ *PID) {
	// Under normal conditions, the method is expected to be called only once.
	// To prevent accidental duplicate responses, we promptly remove the process from the registry
	if r.engine.Registry.remove(r.pid) {
		r.result <- msg
	}
}

func (r *Response) PID() *PID         { return r.pid }
func (r *Response) Shutdown()         {}
func (r *Response) Start()            {}
func (r *Response) Invoke([]Envelope) {}
