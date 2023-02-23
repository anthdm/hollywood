package actor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetByName(t *testing.T) {
	restorePIDSeparator := PIDSeparator
	PIDSeparator = "."

	e := NewEngine()
	e.SpawnFunc(func(c *Context) {}, "foo") // local/foo
	time.Sleep(time.Millisecond * 10)
	proc := e.Registry.getByID("foo")
	expectedPID := NewPID(LocalLookupAddr, "foo")
	require.Equal(t, expectedPID.String(), proc.PID().String())

	// local/foo/bar/q/1
	e.SpawnFunc(func(c *Context) {}, "foo", WithTags("bar", "q", "1"))
	time.Sleep(time.Millisecond * 10)
	proc = e.Registry.getByID("foo.bar.q.1")
	expectedPID = NewPID(LocalLookupAddr, "foo", "bar", "q", "1")
	require.Equal(t, expectedPID.String(), proc.PID().String())

	PIDSeparator = restorePIDSeparator
}
