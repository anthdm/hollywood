package cluster

import (
	"log/slog"
	"math/rand"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type activate struct {
	kind string
	id   string
}

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
	case activate:
		pid := a.activate(NewCID(msg.kind, msg.id))
		c.Respond(pid)
	case *ActivationRequest:
		resp := a.handleActivationRequest(msg)
		c.Respond(resp)
	}
}

func (a *Agent) handleActivationRequest(msg *ActivationRequest) *ActivationResponse {
	if !a.cluster.HasKind(msg.CID.Kind) {
		slog.Error("received activation request but kind not registered on the cluster", "kind", msg.CID.Kind)
		return &ActivationResponse{Success: false}
	}
	kind := a.cluster.kinds[msg.CID.Kind]
	pid := a.cluster.engine.Spawn(kind.producer, msg.CID.ID)
	resp := &ActivationResponse{
		PID:     pid,
		Success: true,
	}
	return resp
}

func (a *Agent) activate(cid *CID) *actor.PID {
	var (
		// TODO: pick member based on rendezvous and custom strategy
		members      = a.members.FilterByKind(cid.Kind)
		owner        = members[rand.Intn(len(members))]
		activatorPID = actor.NewPID(owner.Host, "cluster/"+owner.ID)
	)

	// TODO: topology hash
	req := &ActivationRequest{CID: cid}

	// TODO: retry this couple times
	resp, err := a.cluster.engine.Request(activatorPID, req, time.Millisecond*100).Result()
	if err != nil {
		slog.Error("failed activation request", "err", err)
		return nil
	}
	r, ok := resp.(*ActivationResponse)
	if !ok {

	}
	return r.PID
}

func (a *Agent) handleMembers(members []*Member) {
	joined := NewMemberSet(members...).Except(a.members.Slice())
	left := a.members.Except(members)

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
