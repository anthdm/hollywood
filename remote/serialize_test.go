package remote

import (
	"testing"

	"github.com/anthdm/hollywood/actor"
)

// chmarkSerialize-12    	 8748982	       137.9 ns/op	     144 B/op	       2 allocs/op
func BenchmarkSerialize(b *testing.B) {
	var (
		pid     = actor.NewPID("127.0.0.1:4000", "foo")
		sender  = actor.NewPID("127.0.0.1:8000", "bar")
		payload = &TestMessage{
			Data: []byte("some number of bytes in here would be nice"),
		}
	)

	for i := 0; i < b.N; i++ {
		serialize(pid, sender, payload)
	}
}

// BenchmarkMakeEnvelope/xx-12 	    8151	    148228 ns/op	  155712 B/op	    2050 allocs/op
func BenchmarkMakeEnvelope(b *testing.B) {
	var (
		pid     = actor.NewPID("127.0.0.1:4000", "foo")
		sender  = actor.NewPID("127.0.0.1:8000", "bar")
		payload = &TestMessage{
			Data: []byte("some number of bytes in here would be nice"),
		}
	)

	n := 1024
	streams := make([]writeToStream, n)
	for i := 0; i < n; i++ {
		streams[i] = writeToStream{
			pid:    pid,
			sender: sender,
			msg:    payload,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.Run("xx", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			makeEnvelope(streams)
		}
	})
}
