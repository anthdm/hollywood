package cluster

import (
	"time"

	"github.com/anthdm/hollywood/actor"
)

const memberPingInterval = time.Second * 5

type MemberAddr struct {
	ListenAddr string
	ID         string
}

type memberLeave struct {
	ListenAddr string
}

type memberPing struct{}

type SelfManaged struct {
	cluster        *Cluster
	bootstrapAddrs []MemberAddr
	members        *MemberSet
	memberPinger   actor.SendRepeater
	eventSubPID    *actor.PID

	pid *actor.PID

	membersAlive *MemberSet
}

func NewSelfManagedProvider(addrs ...MemberAddr) Producer {
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &SelfManaged{
				cluster:        c,
				bootstrapAddrs: addrs,
				members:        NewMemberSet(),
				membersAlive:   NewMemberSet(),
			}
		}
	}
}

func (s *SelfManaged) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.pid = c.PID()
		s.members.Add(s.cluster.Member())
		members := &Members{
			Members: s.members.Slice(),
		}
		s.cluster.engine.Send(s.cluster.PID(), members)
		s.memberPinger = c.SendRepeat(c.PID(), memberPing{}, memberPingInterval)
		s.start(c)
	case actor.Stopped:
		s.memberPinger.Stop()
		s.cluster.engine.Unsubscribe(s.eventSubPID)
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
		s.members.ForEach(func(member *Member) bool {
			if member.Host != s.cluster.agentPID.Address {
				ping := &actor.Ping{
					From: c.PID(),
				}
				c.Send(memberToProviderPID(member), ping)
			}
			return true
		})
	case memberLeave:
		member := s.members.GetByHost(msg.ListenAddr)
		s.removeMember(member)
	}
}

// If we receive members from another node in the cluster
// we respond with all the members we know of, and ofcourse
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

func (s *SelfManaged) start(c *actor.Context) {
	s.eventSubPID = c.SpawnChildFunc(func(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case actor.RemoteUnreachableEvent:
			ctx.Send(s.pid, memberLeave{ListenAddr: msg.ListenAddr})
		}
	}, "event")

	s.cluster.engine.Subscribe(s.eventSubPID)

	members := &MembersJoin{
		Members: s.members.Slice(),
	}
	for _, ma := range s.bootstrapAddrs {
		memberPID := actor.NewPID(ma.ListenAddr, "cluster/"+ma.ID+"/provider")
		s.cluster.engine.Send(memberPID, members)
	}
}
