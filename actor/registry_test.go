package actor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetByName(t *testing.T) {
	restorepidSeparator := pidSeparator
	pidSeparator = "."

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

	pidSeparator = restorepidSeparator
}

func TestGetByNameFromContext(t *testing.T) {
	const PidName = "ReceiverFunc"

	e := NewEngine()

	// Receiver staless actor with given name:
	e.SpawnFunc(func(c *Context) {}, PidName)
	time.Sleep(10 * time.Millisecond)

	// Transmitter stateless actor which want to lookup receiver by his name:
	e.SpawnFunc(func(c *Context) {
		pid := c.GetPID(PidName)
		require.NotNil(t, pid)

		expectedPID := NewPID(LocalLookupAddr, PidName)
		require.Equal(t, expectedPID.String(), pid.String())
	}, "TransmitterFunc")
}

func TestGetByNameFromContextWithTags(t *testing.T) {
	const PidName = "ReceiverFunc"

	e := NewEngine()

	// Receiver staless actor with given name:
	e.SpawnFunc(func(c *Context) {}, PidName, WithTags("tag1", "tag2"))
	time.Sleep(10 * time.Millisecond)

	// Transmitter stateless actor which want to lookup receiver by his name:
	e.SpawnFunc(func(c *Context) {
		pid := c.GetPID(PidName, "tag1", "tag2")
		require.NotNil(t, pid)

		expectedPID := NewPID(LocalLookupAddr, PidName, "tag1", "tag2")
		require.Equal(t, expectedPID.String(), pid.String())
	}, "TransmitterFunc")
}
