package cluster

import (
	"log/slog"
	"reflect"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/exp/maps"
)

type (
	activate struct {
		kind   string
		config ActivationConfig
	}
	getMembers struct{}
	getKinds   struct{}
	deactivate struct{ pid *actor.PID }
	getActive  struct {
		id   string
		kind string
	}
)

// Agent is an actor/receiver that is responsible for managing the state
// of the cluster.
type Agent struct {
	members    *MemberSet
	cluster    *Cluster
	kinds      map[string]bool
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
	case actor.Stopped:
	case *ActorTopology:
		a.handleActorTopology(msg)
	case *Members:
		a.handleMembers(msg.Members)
	case *Activation:
		a.handleActivation(msg)
	case activate:
		pid := a.activate(msg.kind, msg.config)
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
		a.handleGetActive(c, msg)
	}
}

func (a *Agent) handleGetActive(c *actor.Context, msg getActive) {
	if len(msg.id) > 0 {
		pid := a.activated[msg.id]
		c.Respond(pid)
	}
	if len(msg.kind) > 0 {
		pids := make([]*actor.PID, 0)
		for id, pid := range a.activated {
			parts := strings.Split(id, "/")
			if len(parts) == 0 {
				break
			}
			kind := parts[0]
			if msg.kind == kind {
				pids = append(pids, pid)
			}
		}
		c.Respond(pids)
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

func (a *Agent) activate(kind string, config ActivationConfig) *actor.PID {
	// Make sure actors are unique across the whole cluster.
	id := kind + "/" + config.id // the id part of the PID
	if _, ok := a.activated[id]; ok {
		slog.Warn("activation failed", "err", "duplicated actor id across the cluster", "id", id)
		return nil
	}
	members := a.members.FilterByKind(kind)
	if len(members) == 0 {
		slog.Warn("could not find any members with kind", "kind", kind)
		return nil
	}
	if config.selectMember == nil {
		config.selectMember = SelectRandomMember
	}
	memberPID := config.selectMember(ActivationDetails{
		Members: members,
		Region:  config.region,
		Kind:    kind,
	})
	if memberPID == nil {
		slog.Warn("activator did not found a member to activate on")
		return nil
	}
	req := &ActivationRequest{Kind: kind, ID: config.id}
	activatorPID := actor.NewPID(memberPID.Host, "cluster/"+memberPID.ID)

	var activationResp *ActivationResponse
	// Local activation
	if memberPID.Host == a.cluster.engine.Address() {
		activationResp = a.handleActivationRequest(req)
	} else {
		// Remote activation
		//
		// TODO: topology hash
		resp, err := a.cluster.engine.Request(activatorPID, req, a.cluster.config.requestTimeout).Result()
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

	slog.Debug("[CLUSTER] member joined",
		"id", member.ID,
		"host", member.Host,
		"kinds", member.Kinds,
		"region", member.Region,
		"members", len(a.members.members))
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

	slog.Debug("[CLUSTER] member left", "id", member.ID, "host", member.Host, "kinds", member.Kinds)
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
		slog.Debug("[CLUSTER] new actor available", "pid", pid)
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
