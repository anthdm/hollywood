package actor

type Started struct{}
type Stopped struct{}

type DeadLetter struct {
	PID     *PID
	Message any
	Sender  *PID
}
