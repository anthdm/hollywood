package actor

import "sync"

// InternalError is a struct used for representing errors within the actor system.
// It encapsulates the source of the error and the error itself.
type InternalError struct {
	From string // From indicates the source or origin of the error.
	Err  error  // Err is the actual error that occurred.
}

// poisonPill is a struct used to signal the shutdown of an actor or process.
// It includes a WaitGroup for synchronization and a flag indicating whether the shutdown is graceful.
type poisonPill struct {
	wg       *sync.WaitGroup // wg is used to wait for the process to finish handling all messages.
	graceful bool            // graceful indicates whether the shutdown should be graceful.
}

type Initialized struct{}
type Started struct{}
type Stopped struct{}
