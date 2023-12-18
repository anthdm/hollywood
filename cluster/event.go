package cluster

// MemberJoinEvent gets triggered each time a new member enters the cluster.
type MemberJoinEvent struct {
	Member *Member
}
