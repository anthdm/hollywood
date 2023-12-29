package ringbuffer

import (
	"sync"
	"sync/atomic"
)

// buffer is a generic type struct that holds the items in the ring buffer
type buffer[T any] struct {
	items           []T   // items holds the elements in the buffer
	head, tail, mod int64 // head and tail are pointers to the start and end of the buffer, mod is the size of the buffer
}

// RingBuffer is a generic type struct that represents a thread-safe ring buffer
type RingBuffer[T any] struct {
	len     int64      // len is the current length of the ring buffer
	content *buffer[T] // content is the actual buffer holding the items
	mu      sync.Mutex // mu is a mutex for ensuring thread safety
}

// New creates a new ring buffer of a given size
func New[T any](size int64) *RingBuffer[T] {
	return &RingBuffer[T]{
		content: &buffer[T]{
			items: make([]T, size),
			head:  0,
			tail:  0,
			mod:   size,
		},
		len: 0,
	}
}

// Push adds an item to the ring buffer, expanding the buffer if necessary
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	rb.content.tail = (rb.content.tail + 1) % rb.content.mod
	if rb.content.tail == rb.content.head { // if the buffer is full
		size := rb.content.mod * 2 // double the size of the buffer
		newBuff := make([]T, size)
		for i := int64(0); i < rb.content.mod; i++ {
			idx := (rb.content.tail + i) % rb.content.mod
			newBuff[i] = rb.content.items[idx]
		}
		content := &buffer[T]{
			items: newBuff,
			head:  0,
			tail:  rb.content.mod,
			mod:   size,
		}
		rb.content = content
	}
	atomic.AddInt64(&rb.len, 1)
	rb.content.items[rb.content.tail] = item
	rb.mu.Unlock()
}

// Len returns the current length of the ring buffer
func (rb *RingBuffer[T]) Len() int64 {
	return atomic.LoadInt64(&rb.len)
}

// Pop removes and returns an item from the ring buffer, if it exists
// Otherwise, it will return a zero value and false, to indicate that it was not returned (e.g. when the buffer is empty)
func (rb *RingBuffer[T]) Pop() (T, bool) {
	if rb.Len() == 0 {
		var t T
		return t, false
	}
	rb.mu.Lock()
	rb.content.head = (rb.content.head + 1) % rb.content.mod
	item := rb.content.items[rb.content.head]
	var t T
	rb.content.items[rb.content.head] = t
	atomic.AddInt64(&rb.len, -1)
	rb.mu.Unlock()
	return item, true
}

// PopN removes and returns 'n' items from the ring buffer.
// It returns a slice of items and a boolean value.
// The boolean value is 'true' if items were successfully removed and 'false' otherwise (e.g., when the buffer is empty).
func (rb *RingBuffer[T]) PopN(n int64) ([]T, bool) {
	if rb.Len() == 0 {
		return nil, false
	}

	rb.mu.Lock()
	content := rb.content

	if n >= rb.len {
		n = rb.len
	}

	atomic.AddInt64(&rb.len, -n)

	items := make([]T, n)
	for i := int64(0); i < n; i++ {
		pos := (content.head + 1 + i) % content.mod
		items[i] = content.items[pos]
		var t T
		content.items[pos] = t
	}

	content.head = (content.head + n) % content.mod
	rb.mu.Unlock()
	return items, true
}
