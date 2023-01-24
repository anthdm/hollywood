package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPID(t *testing.T) {
	address := "127.0.0.1:3000"
	id := "foo"

	pid := NewPID(address, id)
	assert.Equal(t, address+"/"+id, pid.String())

	pid = NewPID(address, id, "100")
	assert.Equal(t, address+"/"+id+"/"+"100", pid.String())

	pid = NewPID(address, id, "100", "bar")
	assert.Equal(t, address+"/"+id+"/100"+"/bar", pid.String())
}

func TestPIDHasTag(t *testing.T) {
	address := "127.0.0.1:3000"
	id := "foo"

	pid := NewPID(address, id)
	assert.False(t, pid.HasTag("bar"))

	pid = NewPID(address, id, "bar")
	assert.True(t, pid.HasTag("bar"))

	pid = NewPID(address, id, "bar", "123")
	assert.True(t, pid.HasTag("bar"))
	assert.True(t, pid.HasTag("123"))
}
