package actor

import (
	"fmt"
	"strings"
)

func NewPID(address, id string, tags ...string) *PID {
	return &PID{
		Address: address,
		ID:      id,
		Tags:    tags,
	}
}

func (pid *PID) String() string {
	if len(pid.Tags) == 0 {
		return fmt.Sprintf("%s/%s", pid.Address, pid.ID)
	}
	tagstr := strings.Join(pid.Tags, "/")
	return fmt.Sprintf("%s/%s/%s", pid.Address, pid.ID, tagstr)
}

func (pid *PID) HasTag(tag string) bool {
	for _, t := range pid.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
