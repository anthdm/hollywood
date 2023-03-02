package actor

import (
	"runtime"

	"github.com/anthdm/hollywood/ggq"
	"github.com/anthdm/hollywood/log"
)

type Inboxer interface {
	Send(Envelope)
	Start(Processer)
	Stop() error
}

type Inbox struct {
	ggq  *ggq.GGQ[Envelope]
	proc Processer
}

func NewInbox(size int) *Inbox {
	in := &Inbox{}
	in.ggq = ggq.New[Envelope](uint32(size), in)
	return in
}

func (in *Inbox) Consume(msgs []Envelope) {
	in.proc.Invoke(msgs)
}

func (in *Inbox) Start(proc Processer) {
	in.proc = proc
	go func() {
		runtime.LockOSThread()
		in.ggq.ReadN()
	}()
	log.Tracew("[INBOX] started", log.M{"pid": proc.PID()})
}

func (in *Inbox) Stop() error {
	in.ggq.Close()
	log.Tracew("[INBOX] closed", log.M{"pid": in.proc.PID()})
	return nil
}

func (in *Inbox) Send(msg Envelope) {
	in.ggq.Write(msg)
}
