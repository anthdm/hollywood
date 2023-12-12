package cluster

import (
	fmt "fmt"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type Agent struct {
	members *MemberSet
	cluster *Cluster
}

func NewAgent(c *Cluster) actor.Producer {
	return func() actor.Receiver {
		return &Agent{
			members: NewMemberSet(),
			cluster: c,
		}
	}
}

func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case *Members:
		a.handleMembers(msg.Members)
	}
}

func (a *Agent) handleMembers(members []*Member) {
	joined := NewMemberSet(members...).Except(a.members.Slice())
	left := a.members.Except(members)

	fmt.Println("joined", joined)
	fmt.Println("left", left)

	for _, member := range joined {
		a.memberJoin(member)
	}
	for _, member := range left {
		a.memberLeave(member)
	}
}

func (a *Agent) memberJoin(member *Member) {
	slog.Info("member joined", "we", a.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
	a.members.Add(member)
	// Send our ActorTopology to this member
}

func (a *Agent) memberLeave(member *Member) {
	slog.Info("member left", "we", a.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
	a.members.Remove(member)
}
