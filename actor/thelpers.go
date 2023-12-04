package actor

import "testing"

// Todo: move this to the test files. It's not used anywhere else.

type TestReceiveFunc func(*testing.T, *Context)

type TestReceiver struct {
	OnReceive TestReceiveFunc
	t         *testing.T
}

func NewTestProducer(t *testing.T, f TestReceiveFunc) Producer {
	return func() Receiver {
		return &TestReceiver{
			OnReceive: f,
			t:         t,
		}
	}
}

func (r *TestReceiver) Receive(ctx *Context) {
	r.OnReceive(r.t, ctx)
}
