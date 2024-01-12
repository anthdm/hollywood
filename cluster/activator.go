package cluster

import (
	"math/rand"
)

// ActivateOnMemberFunc will be invoked by the member that called cluster.Activate.
// Given the ActivationDetails the actor will be spawned on the returned member.
//
// Not that if no member could be selected nil should be returned.
type ActivateOnMemberFunc func(ActivationDetails) *Member

// ActivationDetails holds detailed information about an activation.
type ActivationDetails struct {
	// Region where the actor should be activated on
	Region string
	// A slice of members that are pre-filtered by the kind of the actor
	// that need to be activated
	Members []*Member
	// The kind of the actor
	Kind string
}

// ActivateOnRandomMember returns a random member of the cluster.
func ActivateOnRandomMember(details ActivationDetails) *Member {
	return details.Members[rand.Intn(len(details.Members))]
}
