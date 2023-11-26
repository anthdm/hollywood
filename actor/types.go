package actor

import "sync"

// DeadLetterEvent is broadcasted over the EventStream each time
// a message cannot be delivered to the target PID.
type DeadLetterEvent struct {
	Target  *PID
	Message any
	Sender  *PID
}

// DeadLetterFlush is used to flush the DeadLetter queue.
type DeadLetterFlush struct{}

type DeadLetterFetch struct {
	Flush bool
}

// ActivationEvent is broadcasted over the EventStream each time
// a Receiver is spawned and activated. This mean at the point of
// receiving this event the Receiver is ready to process messages.
type ActivationEvent struct {
	PID *PID
}

// TerminationEvent is broadcasted over the EventStream each time
// a process is terminated.
type TerminationEvent struct {
	PID *PID
}

type InternalError struct {
	From string
	Err  error
}

type poisonPill struct {
	wg *sync.WaitGroup
}
type Initialized struct{}
type Started struct{}
type Stopped struct{}
