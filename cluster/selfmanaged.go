package cluster

import (
	"log/slog"
	"time"

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
	case *Members:
		ourMembers := &Members{
			Members: s.members.ToSlice(),
		}
		c.Respond(ourMembers)
		for _, member := range msg.Members {
			s.addMember(member)
		}
	}
}

// If we receive members from another node in the cluster
// we respond with all the members we know of and, ofcourse
// add the new one.
func (s *SelfManaged) addMember(member *Member) {
	s.members.Add(member)
	slog.Info("got new member", "id", member.ID, "host", member.Host, "kinds", member.Kinds)
}

func (s *SelfManaged) start(c *actor.Context) error {
	members := &Members{
		Members: s.members.ToSlice(),
	}
	for _, m := range s.bootstrapMembers {
		resp, err := s.cluster.engine.Request(memberToProviderPID(m), members, time.Millisecond*100).Result()
		if err != nil {
			slog.Error("provider failed to request members from node", "err", err)
			continue
		}
		members := resp.(*Members)
		for _, member := range members.Members {
			s.addMember(member)
		}
	}
	return nil
}
