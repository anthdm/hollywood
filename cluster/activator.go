package cluster

import (
	"math/rand"
)

// ActivationStrategy is an interface that abstracts the logic on what member of the
// cluster the actor will be activated. Members passed into this function are guaranteed
// to have the given kind locally registered. If no member could be selected nil should be
// returned.
type ActivationStrategy interface {
	ActivateOnMember(ActivationDetails) *Member
}

// ActivationDetails holds detailed information about an activation.
type ActivationDetails struct {
	// Region where the actor should be activated on
	Region string
	// A slice of filtered members by the kind that needs to be activated
	// Members are guaranteed to never by empty.
	Members []*Member
	// The kind that needs to be activated
	Kind string
}

type defaultActivationStrategy struct{}

// DefaultActivationStrategy selects a random member in the cluster.
func DefaultActivationStrategy() defaultActivationStrategy {
	return defaultActivationStrategy{}
}

func (defaultActivationStrategy) ActivateOnMember(details ActivationDetails) *Member {
	return details.Members[rand.Intn(len(details.Members))]
}
