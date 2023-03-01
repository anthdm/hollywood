package actor

import (
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

// func (in *Inbox) Consume(lower, upper int64) {
// 	var (
// 		nmsg   = (upper - lower) + reserv
// 		from   = lower & in.bufferMask
// 		till   = from + nmsg
// 		maxlen = int64(len(in.ringBuffer))
// 	)
// 	if till > maxlen {
// 		msgs := in.ringBuffer[from:maxlen]
// 		in.proc.Invoke(msgs)

// 		trunc := till - maxlen
// 		till = maxlen
// 		msgs = in.ringBuffer[0:trunc]
// 		in.proc.Invoke(msgs)
// 	} else {
// 		msgs := in.ringBuffer[from:till]
// 		in.proc.Invoke(msgs)
// 	}
// }

func (in *Inbox) Start(proc Processer) {
	in.proc = proc
	go in.ggq.ReadN()
	log.Tracew("[INBOX] started", log.M{"pid": proc.PID()})
}

func (in *Inbox) Stop() error {
	defer func() {
		log.Tracew("[INBOX] closed", log.M{"pid": in.proc.PID()})
	}()
	in.ggq.Close()
	return nil
}

func (in *Inbox) Send(msg Envelope) {
	in.ggq.Write(msg)
}
