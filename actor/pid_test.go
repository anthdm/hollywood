package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkXxddd(b *testing.B) {
	var key string
	var keyint uint64
	pid := NewPID("127.0.0.1:4000", "foobar")

	b.Run("+", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key = pid.Address + pidSeparator + pid.ID
		}
	})
	b.Run("key", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			keyint = pid.LookupKey()
		}
	})

	_ = key
	_ = keyint
}

func BenchmarkLookupKey(b *testing.B) {
	pid := NewPID("127.0.0.1:3000", "foo")
	for i := 0; i < b.N; i++ {
		pid.LookupKey()
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
	assert.Equal(t, address+pidSeparator+id, pid.String())

	pid = NewPID(address, id, "100")
	assert.Equal(t, address+pidSeparator+id+pidSeparator+"100", pid.String())

	pid = NewPID(address, id, "100", "bar")
	assert.Equal(t, address+pidSeparator+id+pidSeparator+"100"+pidSeparator+"bar", pid.String())
}
