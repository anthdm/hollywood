package actor

import "fmt"

func NewPID(address, id string) *PID {
	return &PID{
		Address: address,
		ID:      id,
	}
}

func (x *PID) String() string {
	return fmt.Sprintf("%s/%s", x.Address, x.ID)
}
