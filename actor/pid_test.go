package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkNewPID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewPID("127.0.0.1:3000", "foo")
	}
}

func TestPID(t *testing.T) {
	address := "127.0.0.1:3000"
	id := "foo"

	pid := NewPID(address, id)
	assert.Equal(t, address+pidSeparator+id, pid.String())

	pid = NewPID(address, id, "100")
	assert.Equal(t, address+pidSeparator+id+pidSeparator+"100", pid.String())

	pid = NewPID(address, id, "100", "bar")
	assert.Equal(t, address+pidSeparator+id+pidSeparator+"100"+pidSeparator+"bar", pid.String())
}
