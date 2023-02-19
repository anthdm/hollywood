package actor

import (
	"strings"

	"github.com/zeebo/xxh3"
)

var PIDSeparator = "/"

// NewPID returns a new Process ID given an address, name, and optional tags.
// TODO(@anthdm) Can we even optimize this more?
func NewPID(address, id string, tags ...string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
	}
	if len(tags) > 0 {
		p.ID = p.ID + PIDSeparator + strings.Join(tags, PIDSeparator)
	}
	return p
}

func (pid *PID) String() string {
	return pid.Address + PIDSeparator + pid.ID
}

func (pid *PID) Equals(other *PID) bool {
	panic("TODO")
}

func (pid *PID) HasTag(tag string) bool {
	panic("TODO")
}

func (pid *PID) lookupKey() uint64 {
	key := []byte(pid.Address)
	key = append(key, pid.ID...)
	return xxh3.Hash(key)
}
