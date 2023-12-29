package cluster

// MemberSet is a struct that represents a set of members in a cluster.
// It provides functionalities to manage and query the set of members.
type MemberSet struct {
	members map[string]*Member // members is a map of member IDs to Member pointers.
}

// NewMemberSet creates and returns a new MemberSet with the provided members.
func NewMemberSet(members ...*Member) *MemberSet {
	m := make(map[string]*Member)
	for _, member := range members {
		m[member.ID] = member
	}
	return &MemberSet{
		members: m,
	}
}

// Len returns the number of members in the MemberSet.
func (s *MemberSet) Len() int {
	return len(s.members)
}

// GetByHost returns the member with the specified host.
func (s *MemberSet) GetByHost(host string) *Member {
	var theMember *Member
	for _, member := range s.members {
		if member.Host == host {
			theMember = member // Return the member matching the host.
		}
	}
	return theMember
}

// Add adds a new member to the MemberSet.
func (s *MemberSet) Add(member *Member) {
	s.members[member.ID] = member
}

// Contains checks if a member is present in the MemberSet.
func (s *MemberSet) Contains(member *Member) bool {
	_, ok := s.members[member.ID]
	return ok
}

// Remove removes a member from the MemberSet.
func (s *MemberSet) Remove(member *Member) {
	delete(s.members, member.ID)
}

// RemoveByHost removes a member with the specified host from the MemberSet.
func (s *MemberSet) RemoveByHost(host string) {
	member := s.GetByHost(host)
	if member != nil {
		s.Remove(member)
	}
}

// Slice converts the MemberSet to a slice of *Member.
func (s *MemberSet) Slice() []*Member {
	members := make([]*Member, len(s.members))
	i := 0
	for _, member := range s.members {
		members[i] = member
		i++
	}
	return members
}

// ForEach iterates over each member in the MemberSet and applies the given function.
func (s *MemberSet) ForEach(fun func(m *Member) bool) {
	for _, s := range s.members {
		if !fun(s) {
			break
		}
	}
}

// Except returns a slice of members that are in the MemberSet but not in the provided members slice.
func (s *MemberSet) Except(members []*Member) []*Member {
	var (
		except = []*Member{}
		m      = make(map[string]*Member)
	)
	for _, member := range members {
		m[member.ID] = member
	}
	for _, member := range s.members {
		if _, ok := m[member.ID]; !ok {
			except = append(except, member)
		}
	}
	return except
}

// FilterByKind returns a slice of members that have the specified kind.
func (s *MemberSet) FilterByKind(kind string) []*Member {
	members := []*Member{}
	for _, member := range s.members {
		if member.HasKind(kind) {
			members = append(members, member)
		}
	}
	return members
}
