package actor

import (
	"fmt"
	"strings"
)

var PIDSeparator = "/"

func NewPID(address, id string, tags ...string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
		Tags:    tags,
	}
	// Cache the lookup key for the registry for zero allocations
	// when contructing the route.
	p.LookupKey = p.String()
	return p
}

func (pid *PID) String() string {
	if len(pid.Tags) > 0 {
		return fmt.Sprintf("%s%s%s%s%s",
			pid.Address,
			PIDSeparator,
			pid.ID,
			PIDSeparator,
			strings.Join(pid.Tags, PIDSeparator))
	}
	return fmt.Sprintf("%s%s%s", pid.Address, PIDSeparator, pid.ID)
}

func (pid *PID) Equals(other *PID) bool {
	return pid.LookupKey == other.LookupKey
}

func (pid *PID) HasTag(tag string) bool {
	for _, t := range pid.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
