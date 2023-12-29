package actor

import (
	"github.com/zeebo/xxh3"
)

// Constants for the PID separator.
const pidSeparator = "/"

// NewPID is a function that returns a new Process ID given an address and an id.
func NewPID(address, id string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
	}
	return p
}

// String is a method of the PID that returns the string representation of the PID.
func (pid *PID) String() string {
	return pid.Address + pidSeparator + pid.ID
}

// Equals is a method of the PID that compares the PID with another PID for equality.
func (pid *PID) Equals(other *PID) bool {
	return pid.Address == other.Address && pid.ID == other.ID
}

// Child is a method of the PID that creates a child PID based on the given id.
func (pid *PID) Child(id string) *PID {
	childID := pid.ID + pidSeparator + id
	return NewPID(pid.Address, childID)
}

// LookupKey is a method of the PID that returns a hash key for the PID using xxh3 hashing algorithm.
func (pid *PID) LookupKey() uint64 {
	key := []byte(pid.Address)
	key = append(key, pid.ID...)
	return xxh3.Hash(key)
}
