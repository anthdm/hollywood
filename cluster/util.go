package cluster

import "github.com/anthdm/hollywood/actor"

// memberToPID creates a new PID from the member info.
func memberToPID(m *Member) *actor.PID {
	return actor.NewPID(m.Host, "cluster/"+m.ID)
}

// memberToProviderPID create a new PID from the member that
// will target the provider actor of the cluster.
func memberToProviderPID(m *Member) *actor.PID {
	return actor.NewPID(m.Host, "cluster/"+m.ID+"/provider")
}
