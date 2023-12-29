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
	rb.mu.Lock()                                             // lock the mutex to ensure thread safety
	rb.content.tail = (rb.content.tail + 1) % rb.content.mod // move the tail pointer
	if rb.content.tail == rb.content.head {                  // if the buffer is full
		size := rb.content.mod * 2                   // double the size of the buffer
		newBuff := make([]T, size)                   // create a new buffer
		for i := int64(0); i < rb.content.mod; i++ { // copy the items from the old buffer to the new one
			idx := (rb.content.tail + i) % rb.content.mod
			newBuff[i] = rb.content.items[idx]
		}
		content := &buffer[T]{ // create a new buffer struct
			items: newBuff,
			head:  0,
			tail:  rb.content.mod,
			mod:   size,
		}
		rb.content = content // replace the old buffer with the new one
	}
	atomic.AddInt64(&rb.len, 1)              // increment the length of the ring buffer
	rb.content.items[rb.content.tail] = item // add the new item to the buffer
	rb.mu.Unlock()                           // unlock the mutex
}

// Len returns the current length of the ring buffer
func (rb *RingBuffer[T]) Len() int64 {
	return atomic.LoadInt64(&rb.len) // return the length of the ring buffer
}

// Pop removes and returns an item from the ring buffer, if it exists
// Otherwise, it will return a zero value and false, to indicate that it was not returned (e.g. when the buffer is empty)
func (rb *RingBuffer[T]) Pop() (T, bool) {
	if rb.Len() == 0 { // if the buffer is empty
		var t T
		return t, false // return a zero value and false
	}
	rb.mu.Lock()                                             // lock the mutex to ensure thread safety
	rb.content.head = (rb.content.head + 1) % rb.content.mod // move the head pointer
	item := rb.content.items[rb.content.head]                // get the item at the head of the buffer
	var t T
	rb.content.items[rb.content.head] = t // remove the item from the buffer
	atomic.AddInt64(&rb.len, -1)          // decrement the length of the ring buffer
	rb.mu.Unlock()                        // unlock the mutex
	return item, true                     // return the item and true
}

// PopN removes and returns 'n' items from the ring buffer.
// It returns a slice of items and a boolean value.
// The boolean value is 'true' if items were successfully removed and 'false' otherwise (e.g., when the buffer is empty).
func (rb *RingBuffer[T]) PopN(n int64) ([]T, bool) {
	// Check if the ring buffer is empty
	if rb.Len() == 0 {
		// If it is, return nil and false
		return nil, false
	}

	// Lock the mutex to ensure thread safety
	rb.mu.Lock()
	content := rb.content

	// If n is greater than or equal to the length of the ring buffer, set n to the length of the ring buffer
	if n >= rb.len {
		n = rb.len
	}

	// Subtract n from the length of the ring buffer
	atomic.AddInt64(&rb.len, -n)

	// Create a slice to hold the items to be popped
	items := make([]T, n)
	for i := int64(0); i < n; i++ {
		// Calculate the position of the item to be popped
		pos := (content.head + 1 + i) % content.mod
		// Add the item to the slice
		items[i] = content.items[pos]
		// Remove the item from the ring buffer
		var t T
		content.items[pos] = t
	}

	// Move the head pointer
	content.head = (content.head + n) % content.mod

	// Unlock the mutex
	rb.mu.Unlock()

	// Return the items and true
	return items, true
}
