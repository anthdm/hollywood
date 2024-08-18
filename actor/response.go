package actor

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type Response struct {
	engine  *Engine
	pid     *PID
	result  chan any
	timeout time.Duration
	err     error
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

	if r.err != nil {
		return nil, r.err
	}

	select {
	case resp := <-r.result:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Response) Send(_ *PID, msg any, _ *PID) {
	r.result <- msg
}

func (r *Response) PID() *PID                  { return r.pid }
func (r *Response) Shutdown(_ *sync.WaitGroup) {}
func (r *Response) Start()                     {}
func (r *Response) Invoke([]Envelope)          {}
