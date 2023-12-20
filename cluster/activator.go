package cluster

import "math/rand"

// ActivationStrategy is an interface that abstracts the logic on what member of the
// cluster the actor will be activated. Members passed into this function are guaranteed
// to have the given kind locally registered. If no member could be selected nil should be
// returned.
type ActivationStrategy interface {
	SelectMember(members []*Member) *Member
}

// DefaultActivationStrategy selects a random member on the cluster.
type DefaultActivationStrategy struct{}

func (DefaultActivationStrategy) SelectMember(members []*Member) *Member {
	if len(members) == 0 {
		return nil
	}
	return members[rand.Intn(len(members))]
}
