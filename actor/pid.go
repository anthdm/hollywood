package actor

import (
	"fmt"
	"strings"
)

func NewPID(address, id string, tags ...string) *PID {
	p := &PID{
		Address: address,
		ID:      id,
		Tags:    tags,
	}
	if len(p.Tags) > 0 {
		p.ID = p.ID + "/" + strings.Join(p.Tags, "/")
	}
	return p
}

func (pid *PID) String() string {
	return fmt.Sprintf("%s/%s", pid.Address, pid.ID)
}

func (pid *PID) HasTag(tag string) bool {
	for _, t := range pid.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
