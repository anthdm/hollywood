package cluster

import (
	"log/slog"
)

type SelfManaged struct {
	cluster         *Cluster
	bootstrapMember *Member
}

func NewSelfManagedProvider(member ...*Member) *SelfManaged {
	var m *Member
	if len(member) > 0 {
		m = member[0]
	}
	return &SelfManaged{
		bootstrapMember: m,
	}
}

func (s *SelfManaged) Start(c *Cluster) error {
	s.cluster = c

	if s.bootstrapMember == nil {
		return nil
	}

	// Send our member info to one of the known members
	ourMember := s.cluster.Member()
	slog.Info("sending member info", "member", ourMember.ID, "topid", s.bootstrapMember.PID)
	s.cluster.engine.Send(s.bootstrapMember.PID, &MemberJoin{Member: &ourMember})

	return nil
}

func (s *SelfManaged) Stop() error { return nil }

// func NewSelfManaged(members ...Member) Producer {
// 	return func(c *Cluster) actor.Producer {
// 		return func() actor.Receiver {
// 			return &SelfManaged{
// 				cluster:         c,
// 				boostrapMembers: members,
// 			}
// 		}
// 	}
// }

// func (p *SelfManaged) Receive(c *actor.Context) {
// 	switch msg := c.Message().(type) {
// 	case actor.Started:
// 		if len(p.boostrapMembers) > 0 {
// 			for _, member := range p.boostrapMembers {
// 				var (
// 					addr = fmt.Sprintf("%s:%d", member.Host, member.Port)
// 					// TODO: We need access to the configured PIDSeperator
// 					id  = fmt.Sprintf("clusterr/%s/agent", member.ID)
// 					pid = actor.NewPID(addr, id)
// 				)
// 				fmt.Println(pid)
// 				p.cluster.engine.Send(pid, &MemberJoin{})
// 				slog.Debug("attempting bootstrap with", "member", member)
// 			}
// 		}

// 	case *MemberJoin:
// 		slog.Info("new member joined the cluster", "id", msg.ID, "host", msg.Host, "port", msg.Port)
// 	}
// }
