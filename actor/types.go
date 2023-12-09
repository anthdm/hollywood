package actor

import "sync"

type InternalError struct {
	From string
	Err  error
}

type poisonPill struct {
	wg       *sync.WaitGroup
	graceful bool
}
type Initialized struct{}
type Started struct{}
type Stopped struct{}
