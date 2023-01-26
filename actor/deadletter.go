package actor

import (
	reflect "reflect"

	"github.com/anthdm/hollywood/log"
)

type deadLetter struct {
	pid *PID
}

func newDeadLetter() *deadLetter {
	return &deadLetter{
		pid: NewPID(localLookupAddr, "deadLetter"),
	}
}

func (d *deadLetter) PID() *PID { return d.pid }
func (d *deadLetter) Shutdown() {}
func (d *deadLetter) Send(dest *PID, msg any) {
	log.Warnw("[DEADLETTER]", log.M{
		"dest":   dest,
		"msg":    reflect.TypeOf(msg),
		"sender": nil,
	})
}
