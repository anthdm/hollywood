package cluster

import (
	fmt "fmt"
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
)

const memberPingInterval = time.Second * 5

type memberPing struct{}

type SelfManaged struct {
	cluster          *Cluster
	bootstrapMembers []*Member
	members          *MemberSet
	memberPinger     actor.SendRepeater
}

func NewSelfManagedProvider(members ...*Member) Producer {
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &SelfManaged{
				cluster:          c,
				bootstrapMembers: members,
				members:          NewMemberSet(),
			}
		}
	}
}

func (s *SelfManaged) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.members.Add(s.cluster.Member())
		s.memberPinger = c.SendRepeat(c.PID(), memberPing{}, memberPingInterval)
		s.start(c)
	case actor.Stopped:
		s.memberPinger.Stop()
	case *MembersJoin:
		for _, member := range msg.Members {
			s.addMember(member)
		}
		ourMembers := &Members{
			Members: s.members.Slice(),
		}
		s.members.ForEach(func(member *Member) bool {
			s.cluster.engine.Send(memberToProviderPID(member), ourMembers)
			return true
		})
	case *Members:
		for _, member := range msg.Members {
			if !s.members.Contains(member) {
				s.addMember(member)
			}
		}
		if s.members.Len() > 0 {
			members := &Members{
				Members: s.members.Slice(),
			}
			s.cluster.engine.Send(s.cluster.PID(), members)
		}
	case memberPing:
		fmt.Println("pinging all the members", s.members.Len())
		s.members.ForEach(func(member *Member) bool {
			ping := &actor.Ping{
				From: c.PID(),
			}
			_, err := c.Request(memberToProviderPID(member), ping, time.Millisecond*100).Result()
			if err != nil {
				slog.Error("member ping failed", "err", err, "memberID", member.ID)
				s.removeMember(member)
			}
			// TODO: Something is not quite right here!
			// if _, ok := pong.(*actor.Pong); !ok {
			// 	slog.Error("member ping failed", "err", err, "memberID", member.ID)
			// 	s.removeMember(member)
			// }
			return true
		})
	case *actor.Ping:
		// c.Respond(&actor.Pong{From: c.PID()})
	}
}

// If we receive members from another node in the cluster
// we respond with all the members we know of and, ofcourse
// add the new one.
func (s *SelfManaged) addMember(member *Member) {
	if !s.members.Contains(member) {
		s.members.Add(member)
	}
}

func (s *SelfManaged) removeMember(member *Member) {
	if s.members.Contains(member) {
		s.members.Remove(member)
	}
	s.updateCluster()
}

func (s *SelfManaged) updateCluster() {
	members := &Members{
		Members: s.members.Slice(),
	}
	s.cluster.engine.Send(s.cluster.PID(), members)
}

func (s *SelfManaged) start(c *actor.Context) error {
	members := &MembersJoin{
		Members: s.members.Slice(),
	}
	for _, m := range s.bootstrapMembers {
		s.cluster.engine.Send(memberToProviderPID(m), members)
	}
	return nil
}
