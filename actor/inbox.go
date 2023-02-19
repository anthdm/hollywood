package actor

import (
	"github.com/anthdm/hollywood/log"
)

const reserv = 1

type inbox struct {
	ringBuffer []envelope
	bufferMask int64
	proc       *process
}

func newInbox(size int) *inbox {
	return &inbox{
		ringBuffer: make([]envelope, size),
		bufferMask: int64(size) - 1,
	}
}

func (in *inbox) Consume(lower, upper int64) {
	nmsg := (upper - lower) + reserv
	from := lower & in.bufferMask
	till := from + nmsg

	maxlen := int64(len(in.ringBuffer))
	if till > maxlen {
		msgs := in.ringBuffer[from:maxlen]
		in.proc.call(msgs)

		trunc := till - maxlen
		till = maxlen
		msgs = in.ringBuffer[0:trunc]
		in.proc.call(msgs)
	} else {
		msgs := in.ringBuffer[from:till]
		in.proc.call(msgs)
	}
}

func (in *inbox) run(proc *process) {
	in.proc = proc
	go in.proc.disruptor.Reader.Read()
	log.Tracew("[INBOX] started", log.M{})
}

// NOTE: The disruptor will call Close() on the inbox when its fully closed.
func (in *inbox) close() error {
	return in.proc.disruptor.Reader.Close()
}

func (in *inbox) Close() error {
	log.Tracew("[INBOX] closed", log.M{})
	return nil
}

func (in *inbox) send(msg envelope) {
	seq := in.proc.disruptor.Reserve(reserv)
	in.ringBuffer[seq&in.bufferMask] = msg
	in.proc.disruptor.Commit(seq-reserv+1, seq)
}
