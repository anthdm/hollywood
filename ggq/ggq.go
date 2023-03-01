package ggq

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

type Consumer[T any] interface {
	Consume([]T)
}

const cacheLinePadding = 64

const (
	slotEmpty = iota
	slotBusy
	slotCommitted
)

const (
	stateRunning = iota
	stateClosed
)

var (
	state atomic.Int32
)

type slot[T any] struct {
	item T
	atomic.Uint32
}

type GGQ[T any] struct {
	_        [cacheLinePadding]byte
	written  atomic.Uint32
	_        [cacheLinePadding - unsafe.Sizeof(atomic.Uint32{})]byte
	read     atomic.Uint32
	_        [cacheLinePadding - unsafe.Sizeof(atomic.Uint32{})]byte
	buffer   []slot[T]
	_        [cacheLinePadding]byte
	mask     uint32
	consumer Consumer[T]
}

func New[T any](size uint32, consumer Consumer[T]) *GGQ[T] {
	return &GGQ[T]{
		buffer:   make([]slot[T], size),
		mask:     size - 1,
		consumer: consumer,
	}
}

func (q *GGQ[T]) Write(val T) {
	slot := &q.buffer[q.written.Add(1)&q.mask]
	// fmt.Println(slot)
	for !slot.CompareAndSwap(slotEmpty, slotBusy) {
		switch slot.Load() {
		case slotBusy, slotCommitted:
			runtime.Gosched()
		case slotEmpty:
			continue
		}
	}
	slot.item = val
	slot.Store(slotCommitted)
}

func (q *GGQ[T]) ReadN() (T, bool) {
	runtime.LockOSThread()
	var lower, upper uint32
	current := q.read.Load()

	for {
		lower = current + 1
		upper = q.written.Load()

		if lower <= upper {
			// fmt.Printf("L %d U %d\n", lower, upper)
			q.Consume(lower, upper)
			q.read.Store(upper)
			current = upper
			runtime.Gosched()
		} else if upper := q.written.Load(); lower <= upper {
			runtime.Gosched()
		} else if !state.CompareAndSwap(stateClosed, stateRunning) {
			runtime.Gosched()
		} else {
			break
		}
	}
	var t T
	return t, true
}

func (q *GGQ[T]) Consume(lower, upper uint32) {
	lenSlots := upper - lower + 1
	items := make([]T, lenSlots)

	i := 0
	for ; lower <= upper; lower++ {
		slot := &q.buffer[lower&q.mask]
		if slot.CompareAndSwap(slotCommitted, slotBusy) {
			items[i] = slot.item
			slot.Store(slotEmpty)
			i++
		}
	}

	q.consumer.Consume(items[:i])
}

func (q *GGQ[T]) Read() (T, bool) {
	slot := &q.buffer[q.read.Add(1)&q.mask]
	for !slot.CompareAndSwap(slotCommitted, slotBusy) {
		switch slot.Load() {
		case slotBusy:
			runtime.Gosched()
		case slotEmpty:
			if state.CompareAndSwap(stateClosed, stateRunning) {
				var t T
				return t, true
			}
			runtime.Gosched()
		case slotCommitted:
			continue
		}
	}
	item := slot.item
	slot.Store(slotEmpty)
	return item, false
}

func (q *GGQ[T]) Close() {
	state.Store(stateClosed)
}
