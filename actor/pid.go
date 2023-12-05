package actor

import (
	"slices"
	"strings"

	"github.com/zeebo/xxh3"
)

var pidSeparator = "/"

// NewPID returns a new Process ID given an address, name, and optional tags.
// TODO(@anthdm) Can we even optimize this more?
func NewPID(address, id string, tags ...string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
	}
	if len(tags) > 0 {
		p.ID = p.ID + pidSeparator + strings.Join(tags, pidSeparator)
	}
	return p
}

func (pid *PID) String() string {
	return pid.Address + pidSeparator + pid.ID
}

func (pid *PID) Equals(other *PID) bool {
	return pid.Address == other.Address && pid.ID == other.ID
}

func (pid *PID) Child(id string, tags ...string) *PID {
	childID := pid.ID + pidSeparator + id
	if len(tags) == 0 {
		return NewPID(pid.Address, childID)
	}
	return NewPID(pid.Address, childID+pidSeparator+strings.Join(tags, pidSeparator))
}

// HasTag returns whether the provided tag is applied to a pid address or not
func (pid *PID) HasTag(tag string) bool {
	if len(tag) == 0 {
		return false
	}
	parsedPid := strings.Split(pid.ID, pidSeparator)
	return slices.Contains(parsedPid, tag)
}

func (pid *PID) LookupKey() uint64 {
	key := []byte(pid.Address)
	key = append(key, pid.ID...)
	return xxh3.Hash(key)
}
