package actor

import (
	"github.com/anthdm/disruptor"
	"github.com/anthdm/hollywood/log"
)

type Inboxer interface {
	Send(Envelope)
	Start(Processer)
	Stop() error
}

const reserv = 1

type Inbox struct {
	ringBuffer []Envelope
	bufferMask int64
	proc       Processer
	disruptor  disruptor.Disruptor
}

func NewInbox(size int) *Inbox {
	in := &Inbox{
		ringBuffer: make([]Envelope, size),
		bufferMask: int64(size) - 1,
	}
	dis := disruptor.New(
		disruptor.WithCapacity(int64(size)),
		disruptor.WithConsumerGroup(in),
	)
	in.disruptor = dis
	return in
}

func (in *Inbox) Consume(lower, upper int64) {
	var (
		nmsg   = (upper - lower) + reserv
		from   = lower & in.bufferMask
		till   = from + nmsg
		maxlen = int64(len(in.ringBuffer))
	)
	if till > maxlen {
		msgs := in.ringBuffer[from:maxlen]
		in.proc.Invoke(msgs)

		trunc := till - maxlen
		till = maxlen
		msgs = in.ringBuffer[0:trunc]
		in.proc.Invoke(msgs)
	} else {
		msgs := in.ringBuffer[from:till]
		in.proc.Invoke(msgs)
	}
}

func (in *Inbox) Start(proc Processer) {
	in.proc = proc
	go in.disruptor.Reader.Read()
	log.Tracew("[INBOX] started", log.M{"pid": proc.PID()})
}

func (in *Inbox) Stop() error {
	defer func() {
		log.Tracew("[INBOX] closed", log.M{"pid": in.proc.PID()})
	}()
	return in.disruptor.Reader.Close()
}

func (in *Inbox) Send(msg Envelope) {
	seq := in.disruptor.Reserve(reserv)
	in.ringBuffer[seq&in.bufferMask] = msg
	in.disruptor.Commit(seq-reserv+1, seq)
}
