package ringbuffer

import (
	"testing"
)

type Item struct {
	i int
}

func TestPushPop(t *testing.T) {
	rb := New[Item](1024)
	for i := 0; i < 5000; i++ {
		rb.Push(Item{i})
		item, ok := rb.Pop()
		if ok {
			if item.i != i {
				t.Fatal("invalid item popped")
			}
		}
	}
}

func TestPushPopN(t *testing.T) {
	rb := New[Item](1024)
	n := 5000
	for i := 0; i < n; i++ {
		rb.Push(Item{i})
	}
	items, ok := rb.PopN(int64(n))
	if !ok {
		t.Fatal("expected to pop many items")
	}
	for i := 0; i < n; i++ {
		if items[i].i != i {
			t.Fatal("invalid item popped")
		}
	}
}
