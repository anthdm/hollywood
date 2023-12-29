package actor

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// Response is a struct used for handling responses in the actor system.
// It encapsulates the necessary elements for awaiting and processing a response.
type Response struct {
	engine  *Engine       // engine is a reference to the Engine the response is associated with.
	pid     *PID          // pid is a unique identifier for the response.
	result  chan any      // result is a channel used to receive the response message.
	timeout time.Duration // timeout specifies the duration to wait for the response.
}

// NewResponse creates and returns a new Response instance.
func NewResponse(e *Engine, timeout time.Duration) *Response {
	return &Response{
		engine:  e,
		result:  make(chan any, 1), // Initializes the result channel with a buffer size of 1.
		timeout: timeout,
		pid:     NewPID(e.address, "response"+pidSeparator+strconv.Itoa(rand.Intn(math.MaxInt32))),
	}
}

// Result waits for the response message within the specified timeout and returns it.
// If the timeout is exceeded, it returns an error.
func (r *Response) Result() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout) // Create a context with timeout.
	defer func() {
		cancel()                        // Cancel the context when the function returns.
		r.engine.Registry.Remove(r.pid) // Remove the response from the registry after handling.
	}()

	select {
	case resp := <-r.result:
		return resp, nil // Return the response received on the result channel.
	case <-ctx.Done():
		return nil, ctx.Err() // Return nil and an error if the context's deadline is exceeded.
	}
}

// Send places a message into the response's result channel.
func (r *Response) Send(_ *PID, msg any, _ *PID) {
	r.result <- msg
}

// PID returns the unique identifier of the response.
func (r *Response) PID() *PID { return r.pid }

// Shutdown is a placeholder function for the Response, it has no operation.
func (r *Response) Shutdown(_ *sync.WaitGroup) {}

// Start is a placeholder function for the Response, it has no operation.
func (r *Response) Start() {}

// Invoke is a placeholder function for the Response, it has no operation.
func (r *Response) Invoke([]Envelope) {}
