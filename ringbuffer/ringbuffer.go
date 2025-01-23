package ringbuffer

import (
	"sync"
)

type buffer[T any] struct {
	items           []T
	head, tail, mod int64
}

type RingBuffer[T any] struct {
	len     int64
	content *buffer[T]
	mu      sync.Mutex
}

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

func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	nextTail := (rb.content.tail + 1) % rb.content.mod
	if rb.len == rb.content.mod {
		// Buffer is full, advance head to overwrite oldest element
		rb.content.head = (rb.content.head + 1) % rb.content.mod
	} else {
		// Buffer not full, increment length
		rb.len++
	}

	rb.content.tail = nextTail
	rb.content.items[rb.content.tail] = item
	rb.mu.Unlock()
}

func (rb *RingBuffer[T]) Len() int64 {
	rb.mu.Lock()
	length := rb.len
	rb.mu.Unlock()
	return length
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	if rb.len == 0 {
		rb.mu.Unlock()
		var t T
		return t, false
	}
	rb.content.head = (rb.content.head + 1) % rb.content.mod
	item := rb.content.items[rb.content.head]
	var t T
	rb.content.items[rb.content.head] = t
	rb.len--
	rb.mu.Unlock()
	return item, true
}

func (rb *RingBuffer[T]) PopN(n int64) ([]T, bool) {
	rb.mu.Lock()
	if rb.len == 0 {
		rb.mu.Unlock()
		return nil, false
	}
	content := rb.content

	if n >= rb.len {
		n = rb.len
	}
	rb.len -= n

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
