package cluster

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
	mapset "github.com/deckarep/golang-set/v2"
)

type SelfManaged struct {
	cluster          *Cluster
	bootstrapMembers []*Member
	members          mapset.Set[*Member]
}

func NewSelfManagedProvider(members ...*Member) Producer {
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &SelfManaged{
				cluster:          c,
				bootstrapMembers: members,
				members:          mapset.NewSet[*Member](),
			}
		}
	}
}

func (s *SelfManaged) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.members.Add(s.cluster.Member())
		s.start(c)
	case *MembersJoin:
		for _, member := range msg.Members {
			s.addMember(member)
		}
		ourMembers := &Members{
			Members: s.members.ToSlice(),
		}
		for _, member := range s.members.ToSlice() {
			s.cluster.engine.Send(memberToProviderPID(member), ourMembers)
		}

	case *Members:
		for _, member := range msg.Members {
			s.addMember(member)
		}
	}
}

// If we receive members from another node in the cluster
// we respond with all the members we know of and, ofcourse
// add the new one.
func (s *SelfManaged) addMember(member *Member) {
	if !s.containsMember(member) {
		s.members.Add(member)
		slog.Info("got new member", "we", s.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
	}
}

func (s *SelfManaged) containsMember(m *Member) bool {
	for _, member := range s.members.ToSlice() {
		if member.ID == m.ID {
			return true
		}
	}
	return false
}

func (s *SelfManaged) start(c *actor.Context) error {
	members := &MembersJoin{
		Members: s.members.ToSlice(),
	}
	for _, m := range s.bootstrapMembers {
		s.cluster.engine.Send(memberToProviderPID(m), members)
	}
	return nil
}
