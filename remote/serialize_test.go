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
