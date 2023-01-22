package actor

import "fmt"

type PID struct {
	Address string
	ID      string
}

func (p *PID) String() string {
	return fmt.Sprintf("%s/%s", p.Address, p.ID)
}

func NewPID(address, id string) *PID {
	return &PID{
		Address: address,
		ID:      id,
	}
}
