package actor

type Started struct{}

type Stopped struct{}

type WithSender struct {
	Message any
	Sender  *PID
}
