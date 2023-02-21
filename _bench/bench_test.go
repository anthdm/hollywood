package main

import (
	"testing"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

// NOTE: best result with the VTproto compiler
// Normal protoc around 540 ns/op 2 allocs/op
//
// Benchmark/bench_x-12  	  604934	    1857 ns/op	    224 B/op     4 allocs/op
// Benchmark/bench_x-12    	 1410086	   	989 ns/op	    238 B/op	 4 allocs/op
// Benchmark/bench_x-12    	 2403045        564 ns/op	    367 B/op     4 allocs/op
// Benchmark/bench_x-12    	 6422239      	271 ns/op	    180 B/op     2 allocs/op
func Benchmark(b *testing.B) {
	e := actor.NewEngine()
	r := remote.New(e, remote.Config{ListenAddr: "127.0.0.1:5001"})
	e.WithRemote(r)

	pid := actor.NewPID("127.0.0.1:5000", "receiver")

	b.ResetTimer()
	b.ReportAllocs()
	b.Run("bench_x", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			e.Send(pid, pid)
		}
	})
}
