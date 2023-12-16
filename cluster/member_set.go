package cluster

type MemberSet struct {
	members map[string]*Member
}

func NewMemberSet(members ...*Member) *MemberSet {
	m := make(map[string]*Member)
	for _, member := range members {
		m[member.ID] = member
	}
	return &MemberSet{
		members: m,
	}
}

func (s *MemberSet) Len() int {
	return len(s.members)
}

func (s *MemberSet) GetByHost(host string) *Member {
	var theMember *Member
	for _, member := range s.members {
		if member.Host == host {
			theMember = member
		}
	}
	return theMember
}

func (s *MemberSet) Add(member *Member) {
	s.members[member.ID] = member
}

func (s *MemberSet) Contains(member *Member) bool {
	_, ok := s.members[member.ID]
	return ok
}

func (s *MemberSet) Remove(member *Member) {
	delete(s.members, member.ID)
}

func (s *MemberSet) RemoveByHost(host string) {
	member := s.GetByHost(host)
	if member != nil {
		s.Remove(member)
	}
}

func (s *MemberSet) Slice() []*Member {
	members := make([]*Member, len(s.members))
	i := 0
	for _, member := range s.members {
		members[i] = member
		i++
	}
	return members
}

func (s *MemberSet) ForEach(fun func(m *Member) bool) {
	for _, s := range s.members {
		if !fun(s) {
			break
		}
	}
}

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

func (s *MemberSet) FilterByKind(kind string) []*Member {
	members := []*Member{}
	for _, member := range s.members {
		if member.HasKind(kind) {
			members = append(members, member)
		}
	}
	return members
}
