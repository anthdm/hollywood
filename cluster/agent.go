package cluster

import (
	"log/slog"
	"reflect"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/exp/maps"
)

type getActive struct {
	id string
}

type getMembers struct{}

type getKinds struct{}

type activate struct {
	kind   string
	id     string
	region string
}

type deactivate struct {
	pid *actor.PID
}

type Agent struct {
	members *MemberSet
	cluster *Cluster

	kinds map[string]bool

	localKinds map[string]kind

	// All the actors that are available cluster wide.
	activated map[string]*actor.PID
}

func NewAgent(c *Cluster) actor.Producer {
	kinds := make(map[string]bool)
	localKinds := make(map[string]kind)
	for _, kind := range c.kinds {
		kinds[kind.name] = true
		localKinds[kind.name] = kind
	}
	return func() actor.Receiver {
		return &Agent{
			members:    NewMemberSet(),
			cluster:    c,
			kinds:      kinds,
			localKinds: localKinds,
			activated:  make(map[string]*actor.PID),
		}
	}
}

func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case *ActorTopology:
		a.handleActorTopology(msg)
	case *Members:
		a.handleMembers(msg.Members)
	case *Activation:
		a.handleActivation(msg)
	case activate:
		pid := a.activate(msg.kind, msg.id, msg.region)
		c.Respond(pid)
	case deactivate:
		a.bcast(&Deactivation{PID: msg.pid})
	case *Deactivation:
		a.handleDeactivation(msg)
	case *ActivationRequest:
		resp := a.handleActivationRequest(msg)
		c.Respond(resp)
	case getMembers:
		c.Respond(a.members.Slice())
	case getKinds:
		kinds := make([]string, len(a.kinds))
		i := 0
		for kind := range a.kinds {
			kinds[i] = kind
			i++
		}
		c.Respond(kinds)
	case getActive:
		pid := a.activated[msg.id]
		c.Respond(pid)
	}
}

func (a *Agent) handleActorTopology(msg *ActorTopology) {
	for _, actorInfo := range msg.Actors {
		a.addActivated(actorInfo.PID)
	}
}

func (a *Agent) handleDeactivation(msg *Deactivation) {
	a.removeActivated(msg.PID)
	a.cluster.engine.Poison(msg.PID)
	a.cluster.engine.BroadcastEvent(DeactivationEvent{PID: msg.PID})
}

// A new kind is activated on this cluster.
func (a *Agent) handleActivation(msg *Activation) {
	a.addActivated(msg.PID)
	a.cluster.engine.BroadcastEvent(ActivationEvent{PID: msg.PID})
}

func (a *Agent) handleActivationRequest(msg *ActivationRequest) *ActivationResponse {
	if !a.hasKindLocal(msg.Kind) {
		slog.Error("received activation request but kind not registered locally on this node", "kind", msg.Kind)
		return &ActivationResponse{Success: false}
	}
	kind := a.localKinds[msg.Kind]
	pid := a.cluster.engine.Spawn(kind.producer, msg.Kind, actor.WithID(msg.ID))
	resp := &ActivationResponse{
		PID:     pid,
		Success: true,
	}
	return resp
}

func (a *Agent) activate(kind, id, region string) *actor.PID {
	members := a.members.FilterByKind(kind)
	if len(members) == 0 {
		slog.Warn("could not find any members with kind", "kind", kind)
		return nil
	}
	owner := a.cluster.activationStrategy.ActivateOnMember(ActivationDetails{
		Members: members,
		Region:  region,
		Kind:    kind,
	})
	if owner == nil {
		slog.Warn("activator did not found a member to activate on")
		return nil
	}
	req := &ActivationRequest{Kind: kind, ID: id}
	activatorPID := actor.NewPID(owner.Host, "cluster/"+owner.ID)

	var activationResp *ActivationResponse
	// Local activation
	if owner.Host == a.cluster.engine.Address() {
		activationResp = a.handleActivationRequest(req)
	} else {
		// Remote activation

		// TODO: topology hash
		resp, err := a.cluster.engine.Request(activatorPID, req, a.cluster.requestTimeout).Result()
		if err != nil {
			slog.Error("failed activation request", "err", err)
			return nil
		}
		r, ok := resp.(*ActivationResponse)
		if !ok {
			slog.Error("expected *ActivationResponse", "msg", reflect.TypeOf(resp))
			return nil
		}
		if !r.Success {
			slog.Error("activation unsuccessful", "msg", r)
			return nil
		}
		activationResp = r
	}

	a.bcast(&Activation{
		PID: activationResp.PID,
	})

	return activationResp.PID
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
	a.members.Add(member)

	// track cluster wide available kinds
	for _, kind := range member.Kinds {
		if _, ok := a.kinds[kind]; !ok {
			a.kinds[kind] = true
		}
	}

	actorInfos := make([]*ActorInfo, 0)
	for _, pid := range a.activated {
		actorInfo := &ActorInfo{
			PID: pid,
		}
		actorInfos = append(actorInfos, actorInfo)
	}

	// Send our ActorTopology to this member
	if len(actorInfos) > 0 {
		a.cluster.engine.Send(member.PID(), &ActorTopology{Actors: actorInfos})
	}

	// Broadcast MemberJoinEvent
	a.cluster.engine.BroadcastEvent(MemberJoinEvent{
		Member: member,
	})

	slog.Debug("member joined", "id", member.ID, "host", member.Host, "kinds", member.Kinds, "region", member.Region)
}

func (a *Agent) memberLeave(member *Member) {
	a.members.Remove(member)
	a.rebuildKinds()

	// Remove all the activeKinds that where running on the member that left the cluster.
	for _, pid := range a.activated {
		if pid.Address == member.Host {
			a.removeActivated(pid)
		}
	}

	a.cluster.engine.BroadcastEvent(MemberLeaveEvent{Member: member})

	slog.Debug("member left", "id", member.ID, "host", member.Host, "kinds", member.Kinds)
}

func (a *Agent) bcast(msg any) {
	a.members.ForEach(func(member *Member) bool {
		a.cluster.engine.Send(member.PID(), msg)
		return true
	})
}

func (a *Agent) addActivated(pid *actor.PID) {
	if _, ok := a.activated[pid.ID]; !ok {
		a.activated[pid.ID] = pid
		slog.Debug("new actor available on cluster", "pid", pid)
	}
}

func (a *Agent) removeActivated(pid *actor.PID) {
	delete(a.activated, pid.ID)
	slog.Debug("actor removed from cluster", "pid", pid)
}

func (a *Agent) hasKindLocal(name string) bool {
	_, ok := a.localKinds[name]
	return ok
}

func (a *Agent) rebuildKinds() {
	maps.Clear(a.kinds)
	a.members.ForEach(func(m *Member) bool {
		for _, kind := range m.Kinds {
			if _, ok := a.kinds[kind]; !ok {
				a.kinds[kind] = true
			}
		}
		return true
	})
}
