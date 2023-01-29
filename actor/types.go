package actor

// DeadLetterEvent is broadcasted over the EventStream each time
// a message cannot be delivered to the target PID.
type DeadLetterEvent struct {
	Target  *PID
	Message any
	Sender  *PID
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

type SpawnEvent struct {
}

type InternalError struct {
	From string
	Err  error
}

type Initialized struct{}

type Started struct{}

type Stopped struct{}
