package safemap

import (
	"math/rand"
	"testing"
)

func TestNewAndSet(t *testing.T) {
	sm := New[int, string]()
	sm.Set(1, "one")
	sm.Set(2, "two")

	if sm.Len() != 2 {
		t.Errorf("Expected length 2, got %d", sm.Len())
	}
}

func TestGet(t *testing.T) {
	sm := New[int, string]()
	sm.Set(1, "one")

	val, ok := sm.Get(1)
	if !ok || val != "one" {
		t.Errorf("Expected 'one', got %s", val)
	}
	_, ok = sm.Get(2)
	if ok {
		t.Errorf("Expected false, got true")
	}
}

func TestDelete(t *testing.T) {
	sm := New[int, string]()
	sm.Set(1, "one")
	sm.Delete(1)
	_, ok := sm.Get(1)
	if ok {
		t.Errorf("Expected key 1 to be deleted")
	}

	sm.Delete(2) // just make sure this doesn't panic
}

func TestLen(t *testing.T) {
	sm := New[int, string]()
	sm.Set(1, "one")
	sm.Set(2, "two")
	sm.Set(3, "three")
	sm.Delete(2)

	if sm.Len() != 2 {
		t.Errorf("Expected length 2, got %d", sm.Len())
	}
}

func TestForEach(t *testing.T) {
	sm := New[int, string]()
	sm.Set(1, "one")
	sm.Set(2, "two")

	keys := make([]int, 0)
	sm.ForEach(func(k int, v string) {
		keys = append(keys, k)
	})

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Check if keys 1 and 2 are present
	if !contains(keys, 1) || !contains(keys, 2) {
		t.Errorf("Expected keys 1 and 2, got %v", keys)
	}
}

// Helper function to check if a slice contains a specific element.
func contains(slice []int, element int) bool {
	for _, a := range slice {
		if a == element {
			return true
		}
	}
	return false
}

// Benchmark Get
func BenchmarkGetConcurrent(b *testing.B) {
	ds := New[uint64, uint64]()
	// Pre-fill the data store with test data if needed
	for i := 0; i < 100000; i++ {
		ds.Set(uint64(i), uint64(i))
	}
	b.SetParallelism(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := rand.Uint64() % 100000
			_, _ = ds.Get(r)
		}
	})
}

// fool the compiler
var silly uint64

// Benchmark Set
func BenchmarkSetConcurrent(b *testing.B) {
	ds := New[uint64, uint64]()
	b.SetParallelism(100)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := rand.Uint64() % 100000
			silly, _ = ds.Get(r)
			ds.Set(r, r)
			silly, _ = ds.Get(r)

		}
	})
}
