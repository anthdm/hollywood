package safemap_test

import (
	"github.com/anthdm/hollywood/safemap"
	"math/rand"
	"testing"
)

// Benchmark Get
func BenchmarkGetConcurrent(b *testing.B) {
	ds := safemap.New[uint64, uint64]()
	// Pre-fill the data store with test data if needed
	for i := 0; i < 100000; i++ {
		ds.Set(uint64(i), uint64(i))
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := rand.Uint64() % 100000
			_, _ = ds.Get(r)
		}
	})
}

// Benchmark Set
func BenchmarkSetConcurrent(b *testing.B) {
	ds := safemap.New[uint64, uint64]()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := rand.Uint64() % 100000
			ds.Set(r, r)
		}
	})
}
