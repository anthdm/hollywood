package actor

import (
	"github.com/zeebo/xxh3"
)

const pidSeparator = "/"

// NewPID returns a new Process ID given an address and an id.
func NewPID(address, id string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
	}
	return p
}

func (pid *PID) String() string {
	return pid.Address + pidSeparator + pid.ID
}

func (pid *PID) Equals(other *PID) bool {
	return pid.Address == other.Address && pid.ID == other.ID
}

func (pid *PID) Child(id string) *PID {
	childID := pid.ID + pidSeparator + id
	return NewPID(pid.Address, childID)
}

func (pid *PID) LookupKey() uint64 {
	key := []byte(pid.Address)
	key = append(key, pid.ID...)
	return xxh3.Hash(key)
}
