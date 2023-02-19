package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkLookupKey(b *testing.B) {
	pid := NewPID("127.0.0.1:3000", "foo")
	for i := 0; i < b.N; i++ {
		pid.lookupKey()
	}
}

func BenchmarkNewPID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewPID("127.0.0.1:3000", "foo")
	}
}

func TestPID(t *testing.T) {
	address := "127.0.0.1:3000"
	id := "foo"

	pid := NewPID(address, id)
	assert.Equal(t, address+PIDSeparator+id, pid.String())

	pid = NewPID(address, id, "100")
	assert.Equal(t, address+PIDSeparator+id+PIDSeparator+"100", pid.String())

	pid = NewPID(address, id, "100", "bar")
	assert.Equal(t, address+PIDSeparator+id+PIDSeparator+"100"+PIDSeparator+"bar", pid.String())
}
