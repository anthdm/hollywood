package ggq

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/anthdm/hollywood/log"
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

type slot[T any] struct {
	item T
	atomic.Uint32
}

type GGQ[T any] struct {
	_          [cacheLinePadding]byte
	written    atomic.Uint32
	_          [cacheLinePadding - unsafe.Sizeof(atomic.Uint32{})]byte
	read       atomic.Uint32
	_          [cacheLinePadding - unsafe.Sizeof(atomic.Uint32{})]byte
	state      atomic.Uint32
	_          [cacheLinePadding - unsafe.Sizeof(atomic.Uint32{})]byte
	isIdling   atomic.Bool
	buffer     []slot[T]
	_          [cacheLinePadding]byte
	mask       uint32
	consumer   Consumer[T]
	itemBuffer []T
	cond       *sync.Cond
}

func New[T any](size uint32, consumer Consumer[T]) *GGQ[T] {
	if !isPOW2(size) {
		log.Fatalw("the size of the queue need to be a number that is the power of 2", log.M{})
	}
	return &GGQ[T]{
		buffer:     make([]slot[T], size),
		mask:       size - 1,
		consumer:   consumer,
		itemBuffer: make([]T, size+1),
		cond:       sync.NewCond(nil),
	}
}

func (q *GGQ[T]) Write(val T) {
	slot := &q.buffer[q.written.Add(1)&q.mask]
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
	var lower, upper uint32
	current := q.read.Load()
	for {
		lower = current + 1
		upper = q.written.Load()
		if lower <= upper {
			q.Consume(lower, upper)
			q.read.Store(upper)
			current = upper
			runtime.Gosched()
		} else if upper := q.written.Load(); lower <= upper {
			runtime.Gosched()
		} else if !q.state.CompareAndSwap(stateClosed, stateRunning) {
			var mu sync.Mutex
			q.cond.L = &mu
			q.isIdling.Store(true)
			mu.Lock()
			q.cond.Wait()
			mu.Unlock()
			q.isIdling.Store(false)
		} else {
			break
		}
	}
	var t T
	return t, true
}

// Awake the queue if its in the idle state.
func (q *GGQ[T]) Awake() {
	if q.isIdling.Load() {
		q.cond.Signal()
	}
}

func (q *GGQ[T]) IsIdle() bool {
	return q.isIdling.Load()
}

func (q *GGQ[T]) Consume(lower, upper uint32) {
	consumed := 0
	for ; lower <= upper; lower++ {
		slot := &q.buffer[lower&q.mask]
		for !slot.CompareAndSwap(slotCommitted, slotBusy) {
			switch slot.Load() {
			case slotBusy:
				runtime.Gosched()
			case slotCommitted:
				continue
			}
		}
		q.itemBuffer[consumed] = slot.item
		slot.Store(slotEmpty)
		consumed++
	}
	q.consumer.Consume(q.itemBuffer[:consumed])
}

// ReadN gives way better performance, due to batching messages with
// lock os thread.
func (q *GGQ[T]) Read() (T, bool) {
	slot := &q.buffer[q.read.Add(1)&q.mask]
	for !slot.CompareAndSwap(slotCommitted, slotBusy) {
		switch slot.Load() {
		case slotBusy:
			runtime.Gosched()
		case slotEmpty:
			if q.state.CompareAndSwap(stateClosed, stateRunning) {
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
	q.state.Store(stateClosed)
	q.cond.Signal()
}

func isPOW2(n uint32) bool {
	if n <= 0 {
		return false
	}
	return (n & (n - 1)) == 0
}
