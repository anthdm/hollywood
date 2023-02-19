package remote

import (
	"time"
)

type batch struct {
	items    []writeToStream
	timeout  time.Duration
	size     int
	timer    *time.Timer
	sendFunc func([]writeToStream)
}

func newBatch(fn func([]writeToStream)) *batch {
	return &batch{
		timeout:  time.Millisecond * 5,
		items:    make([]writeToStream, 1024),
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

func (b *batch) add(msg writeToStream) {
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
