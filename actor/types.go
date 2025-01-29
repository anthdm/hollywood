package actor

import "context"

type InternalError struct {
	From string
	Err  error
}

type poisonPill struct {
	cancel   context.CancelFunc
	graceful bool
}
type Initialized struct{}
type Started struct{}
type Stopped struct{}
