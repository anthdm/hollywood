package actor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetByName(t *testing.T) {
	e := NewEngine()
	e.SpawnFunc(func(c *Context) {}, "foo") // local/foo
	time.Sleep(time.Millisecond * 10)
	proc := e.registry.getByName("foo")
	require.Equal(t, "local/foo", proc.PID().String())

	// local/foo/bar/q/1
	e.SpawnFunc(func(c *Context) {}, "foo", WithTags("bar", "q", "1"))
	time.Sleep(time.Millisecond * 10)
	proc = e.registry.getByName("foo/bar/q/1")
	require.Equal(t, "local/foo/bar/q/1", proc.PID().String())
}
