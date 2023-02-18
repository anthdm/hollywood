package actor

import (
	"github.com/anthdm/hollywood/log"
)

const reserv = 1

type inbox struct {
	ringBuffer []envelope
	bufferMask int
	proc       *process
}

func newInbox(size int) *inbox {
	return &inbox{
		ringBuffer: make([]envelope, size),
		bufferMask: size - 1,
	}
}

func (in *inbox) Consume(lower, upper int64) {
	for ; lower <= upper; lower++ {
		message := in.ringBuffer[lower&int64(in.bufferMask)]
		in.proc.call(message)
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
	in.ringBuffer[seq&int64(in.bufferMask)] = msg
	in.proc.disruptor.Commit(seq-reserv+1, seq)
}
