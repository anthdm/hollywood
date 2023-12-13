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

// NewCID returns a new Cluster ID.
func NewCID(kind, id string) *CID {
	return &CID{
		Kind: kind,
		ID:   id,
	}
}

// TODO: Maybe relocate this function.
// HasKind returns true whether the Member has the given kind registered.
func (m *Member) HasKind(kind string) bool {
	for _, k := range m.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}
