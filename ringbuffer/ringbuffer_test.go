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

func TestPopNInto(t *testing.T) {
	rb := New[Item](1024)

	// Push more than initial capacity to exercise growth path too.
	n := 5000
	for i := 0; i < n; i++ {
		rb.Push(Item{i})
	}

	// Reusable destination buffer (should not be replaced once large enough)
	dst := make([]Item, 0, 64)

	// Pop in chunks
	outAll := make([]Item, 0, n)
	for len(outAll) < n {
		want := int64(123)                // arbitrary batch size
		usedBefore := &dst[0:cap(dst)][0] // pointer into backing array (safe because cap>0)

		msgs, ok := rb.PopNInto(dst, want)
		if !ok || len(msgs) == 0 {
			t.Fatalf("expected more items, got ok=%v len=%d", ok, len(msgs))
		}

		// Ensure PopNInto reuses the backing array when cap(dst) is sufficient.
		// It may grow dst during early iterations until cap >= want; after that it must be stable.
		if cap(dst) >= int(want) {
			usedAfter := &msgs[0:cap(msgs)][0]
			if usedAfter != usedBefore {
				t.Fatal("expected PopNInto to reuse dst backing array when capacity is sufficient")
			}
		}

		outAll = append(outAll, msgs...)

		// Reset dst for reuse (same backing array)
		dst = msgs[:0]
	}

	// Validate ordering
	for i := 0; i < n; i++ {
		if outAll[i].i != i {
			t.Fatalf("invalid item popped at %d: got %d want %d", i, outAll[i].i, i)
		}
	}

	// Now buffer is empty
	msgs, ok := rb.PopNInto(dst, 1)
	if ok || len(msgs) != 0 {
		t.Fatalf("expected empty pop: ok=%v len=%d", ok, len(msgs))
	}
}

func TestPopNIntoThreadSafety(t *testing.T) {
	t.Run("PopNInto should be thread-safe", func(t *testing.T) {
		testCase := func() {
			rb := New[int](1024)
			rb.Push(1)

			counter := atomic.Int32{}
			wg := sync.WaitGroup{}
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					// Each goroutine uses its own dst to avoid sharing memory.
					dst := make([]int, 0, 1)
					_, ok := rb.PopNInto(dst, 1)
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

		for i := 0; i < 100_000; i++ {
			testCase()
		}
	})
}
