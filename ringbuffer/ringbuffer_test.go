package ringbuffer

import (
	"sync"
	"sync/atomic"
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

func TestPopThreadSafety(t *testing.T) {
	t.Run("Pop should be thread-safe", func(t *testing.T) {
		testCase := func() {
			rb := New[int](4)
			rb.Push(1)
			wg := sync.WaitGroup{}
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rb.Pop()
				}()
			}
			wg.Wait()
			if rb.Len() == -1 {
				t.Fatal("item popped twice")
			}
		}

		// Increase the number of iterations to raise the likelihood of reproducing the race condition
		for i := 0; i < 100_000; i++ {
			testCase()
		}
	})

	t.Run("PopN should be thread-safe", func(t *testing.T) {
		testCase := func() {
			rb := New[int](4)
			rb.Push(1)
			counter := atomic.Int32{}
			wg := sync.WaitGroup{}
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, ok := rb.PopN(1)
					if ok {
						counter.Add(1)
					}
				}()
			}
			wg.Wait()
			if counter.Load() > 1 {
				t.Fatal("false positive item removal")
			}
		}

		// Increase the number of iterations to raise the likelihood of reproducing the race condition
		for i := 0; i < 100_000; i++ {
			testCase()
		}
	})
}
