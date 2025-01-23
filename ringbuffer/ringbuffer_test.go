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
	size := int64(1024)
	rb := New[Item](size)
	n := 5000

	// Since we're pushing more items than buffer size,
	// only the last 'size' items should be present
	for i := 0; i < n; i++ {
		rb.Push(Item{i})
	}

	items, ok := rb.PopN(size)
	if !ok {
		t.Fatal("expected to pop items")
	}

	// Verify we got the most recent items (n-size to n)
	startIdx := n - int(size)
	for i := 0; i < len(items); i++ {
		expected := startIdx + i
		if items[i].i != expected {
			t.Fatalf("invalid item at position %d: got %d, want %d",
				i, items[i].i, expected)
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

func TestPushPopReal(t *testing.T) {
	rb := New[int](10)

	// Push 10 values
	for i := int64(0); i < 10; i++ {
		rb.Push(int(i))
		if rb.Len() != i+1 {
			t.Fatalf("expected length %d after push, got %d", i+1, rb.Len())
		}
	}

	// Pop 10 values
	for i := int64(0); i < 10; i++ {
		val, ok := rb.Pop()
		if !ok {
			t.Fatalf("expected Pop to succeed on iteration %d", i)
		}
		if val != int(i) {
			t.Errorf("expected value %d, got %d", i, val)
		}
	}

	// Verify buffer is empty
	if rb.Len() != 0 {
		t.Errorf("expected empty buffer, got length %d", rb.Len())
	}
}

func TestPopEdge(t *testing.T) {
	rb := New[int](1)

	// Test pop from empty buffer
	item, ok := rb.Pop()
	if ok {
		t.Fatalf("expected Pop to return false on empty buffer, got true")
	}
	if item != 0 {
		t.Errorf("expected zero value for item, got %d", item)
	}
}

func TestPopNEdge(t *testing.T) {
	rb := New[int](1)

	// Test PopN on empty buffer
	items, ok := rb.PopN(1)
	if ok {
		t.Fatalf("expected PopN to return false on empty buffer, got true")
	}
	if items != nil {
		t.Errorf("expected nil slice, got %v", items)
	}

	// Test PopN with n <= 0
	items, ok = rb.PopN(0)
	if ok || items != nil {
		t.Errorf("expected PopN(0) to return (nil, false) on empty buffer, got (%v, %v)", items, ok)
	}

	// Test PopN with n > buffer size
	rb.Push(1)
	items, ok = rb.PopN(2)
	if !ok || len(items) != 1 || items[0] != 1 {
		t.Errorf("expected PopN(2) to return slice [1], got %v, %v", items, ok)
	}
}

func TestCircularWrap(t *testing.T) {
	rb := New[int](3)

	// Fill buffer
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	// Pop one and push one to force wrap
	val, _ := rb.Pop()
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
	rb.Push(4)

	// Verify correct order after wrap
	expected := []int{2, 3, 4}
	for i, exp := range expected {
		val, ok := rb.Pop()
		if !ok || val != exp {
			t.Errorf("item %d: expected %d, got %d", i, exp, val)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	rb := New[int](5)
	done := make(chan bool)

	// Concurrent push
	go func() {
		for i := 0; i < 100; i++ {
			rb.Push(i)
		}
		done <- true
	}()

	// Concurrent pop
	go func() {
		for i := 0; i < 100; i++ {
			rb.Pop()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

func BenchmarkPush(b *testing.B) {
	rb := New[int](100)
	for i := 0; i < b.N; i++ {
		rb.Push(i)
	}
}

func BenchmarkPop(b *testing.B) {
	rb := New[int](100)
	for i := 0; i < 100; i++ {
		rb.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Pop()
		rb.Push(i)
	}
}

func BenchmarkPopN(b *testing.B) {
	rb := New[int](100)
	for i := 0; i < 100; i++ {
		rb.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.PopN(10)
		for j := 0; j < 10; j++ {
			rb.Push(j)
		}
	}
}
