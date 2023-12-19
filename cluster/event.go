package cluster

import "github.com/anthdm/hollywood/actor"

// MemberJoinEvent gets triggered each time a new member enters the cluster.
type MemberJoinEvent struct {
	Member *Member
}

// MemberLeaveEvent gets triggered each time a member leaves the cluster.
type MemberLeaveEvent struct {
	Member *Member
}

// ActivationEvent gets triggered each time a new actor is activated somewhere on
// the cluster.
type ActivationEvent struct {
	PID *actor.PID
}

// DeactivationEvent gets triggered each time an actor gets deactivated somewhere on
// the cluster.
type DeactivationEvent struct {
	PID *actor.PID
}
