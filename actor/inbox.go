package actor

import (
	"github.com/anthdm/disruptor"
	"github.com/anthdm/hollywood/log"
)

type Inboxer interface {
	Send(envelope)
	Start(processer)
	Close() error
}

const reserv = 1

type inbox struct {
	ringBuffer []envelope
	bufferMask int64
	proc       processer
	disruptor  disruptor.Disruptor
}

func newInbox(size int) *inbox {
	in := &inbox{
		ringBuffer: make([]envelope, size),
		bufferMask: int64(size) - 1,
	}
	dis := disruptor.New(
		disruptor.WithCapacity(int64(size)),
		disruptor.WithConsumerGroup(in),
	)
	in.disruptor = dis
	return in
}

func (in *inbox) Consume(lower, upper int64) {
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

func (in *inbox) Start(proc processer) {
	in.proc = proc
	go in.disruptor.Reader.Read()
	log.Tracew("[INBOX] started", log.M{})
}

func (in *inbox) Close() error {
	defer func() {
		log.Tracew("[INBOX] closed", log.M{})
	}()
	return in.disruptor.Reader.Close()
}

func (in *inbox) Send(msg envelope) {
	seq := in.disruptor.Reserve(reserv)
	in.ringBuffer[seq&in.bufferMask] = msg
	in.disruptor.Commit(seq-reserv+1, seq)
}
