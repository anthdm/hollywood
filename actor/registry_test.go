package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fooProc struct {
}

func (p fooProc) Start()               {}
func (p fooProc) PID() *PID            { return NewPID(LocalLookupAddr, "foo") }
func (p fooProc) Send(*PID, any, *PID) {}
func (p fooProc) Invoke([]Envelope)    {}
func (p fooProc) Shutdown()            {}

func TestGetRemoveAdd(t *testing.T) {
	e, _ := NewEngine(NewEngineConfig())
	reg := newRegistry(e)
	eproc := fooProc{}
	reg.add(eproc)
	proc := reg.getByID(eproc.PID().ID)
	assert.Equal(t, proc, eproc)
	proc = reg.get(eproc.PID())
	assert.Equal(t, proc, eproc)
	reg.Remove(eproc.PID())
	proc = reg.get(eproc.PID())
	assert.Nil(t, proc)
}
