package cluster

import (
	fmt "fmt"
	"math"
	"math/rand"
)

// ActivationConfig...
type ActivationConfig struct {
	id           string
	region       string
	selectMember SelectMemberFunc
}

// NewActivationConfig returns a new default config.
func NewActivationConfig() ActivationConfig {
	return ActivationConfig{
		id:           fmt.Sprintf("%d", rand.Intn(math.MaxInt)),
		region:       "default",
		selectMember: SelectRandomMember,
	}
}

// WithSelectMemberFunc set's the fuction that will be invoked during
// the activation process.
// It will select the member where the actor will be activated/spawned on.
func (config ActivationConfig) WithSelectMemberFunc(fun SelectMemberFunc) ActivationConfig {
	config.selectMember = fun
	return config
}

// WithID set's the id of the actor that will be activated on the cluster.
//
// Defaults to a random identifier.
func (config ActivationConfig) WithID(id string) ActivationConfig {
	config.id = id
	return config
}

// WithRegion set's the region on where this actor should be spawned.
//
// Defaults to a "default".
func (config ActivationConfig) WithRegion(region string) ActivationConfig {
	config.region = region
	return config
}

// SelectMemberFunc will be invoked during the activation process.
// Given the ActivationDetails the actor will be spawned on the returned member.
type SelectMemberFunc func(ActivationDetails) *Member

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

// SelectRandomMember selects a random member of the cluster.
func SelectRandomMember(details ActivationDetails) *Member {
	return details.Members[rand.Intn(len(details.Members))]
}
