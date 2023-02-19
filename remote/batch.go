package remote

import (
	"time"
)

type batch struct {
	items    []*streamDeliver
	timeout  time.Duration
	size     int
	timer    *time.Timer
	sendFunc func([]*streamDeliver)
}

func newBatch(fn func([]*streamDeliver)) *batch {
	return &batch{
		timeout:  time.Millisecond * 5,
		items:    make([]*streamDeliver, 1024),
		sendFunc: fn,
	}
}

func (b *batch) flush() {
	// log.Tracew("flushed batch", log.M{
	// 	"flushed": b.size,
	// 	"max":     len(b.items),
	// })
	b.sendFunc(b.items[:b.size])
	b.size = 0
}

func (b *batch) add(msg *streamDeliver) {
	b.items[b.size] = msg
	b.size++

	if b.size >= len(b.items) {
		b.flush()
	} else {
		if b.timer == nil {
			b.timer = time.AfterFunc(b.timeout, b.flush)
		} else {
			b.timer.Reset(b.timeout)
		}
	}
}
