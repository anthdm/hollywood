package actor

import (
	reflect "reflect"

	"github.com/anthdm/hollywood/log"
)

type deadletter struct{}

func newDeadletter() Receiver {
	return &deadletter{}
}

func (d *deadletter) Receive(ctx *Context) {
	if msg, ok := ctx.Message().(*DeadLetter); ok {
		log.Warnw("[DEADLETTER]", log.M{
			"dest":   msg.PID,
			"msg":    reflect.TypeOf(msg.Message),
			"sender": nil,
		})
	}
}
