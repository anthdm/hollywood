package actor

import (
	"testing"
)

type dummyProc struct{}

func (dummyProc) PID() *PID {
	return NewPID("127.0.0.1:2000", "foo", "bar")
}
func (dummyProc) Send(any) {
}
func (dummyProc) Shutdown() {}

func TestRegistry(t *testing.T) {
	// var (
	// 	r     = newRegistry()
	// 	pid   = NewPID(localAddrLookup, "foo")
	// 	dummy = &dummyProc{}
	// )
}

func BenchmarkXxx(b *testing.B) {

}

func BenchmarkKeyWriter(b *testing.B) {
	w := newKeyWriter()
	pid := NewPID("foo", "bar")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.writePIDKey(pid)
	}
	b.StopTimer()
}
