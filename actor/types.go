package actor

import "sync"

// DeadLetterEvent is broadcasted over the EventStream each time
// a message cannot be delivered to the target PID.
type DeadLetterEvent struct {
	Target  *PID
	Message any
	Sender  *PID
}

// ActorStartedEvent is broadcasted over the EventStream each time
// a Receiver (Actor) is spawned and activated. This means, that at
// the point of receiving this event the Receiver (Actor) is ready
// to process messages.
type ActorStartedEvent struct {
	PID *PID
}

// ActorStoppedEvent is broadcasted over the EventStream each time
// a process is terminated.
type ActorStoppedEvent struct {
	PID *PID
}

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
