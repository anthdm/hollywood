package cluster

import (
	"log/slog"
	"math/rand"
	reflect "reflect"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type getKinds struct{}

type activate struct {
	kind string
	id   string
}

type deactivate struct {
	cid *CID
}

type Agent struct {
	members *MemberSet
	cluster *Cluster

	kinds map[string]bool

	localKinds map[string]Kind

	// all the active kinds on the cluster
	activeKinds *KindLookup
}

func NewAgent(c *Cluster) actor.Producer {
	kinds := make(map[string]bool)
	localKinds := make(map[string]Kind)
	for _, kind := range c.kinds {
		kinds[kind.name] = true
		localKinds[kind.name] = kind
	}
	return func() actor.Receiver {
		return &Agent{
			members:     NewMemberSet(),
			cluster:     c,
			activeKinds: NewKindLookup(),
			kinds:       kinds,
			localKinds:  localKinds,
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
		cid := a.activate(msg.kind, msg.id)
		c.Respond(cid)
	case deactivate:
		// TODO:
		a.bcast(&Deactivation{})
	case *ActivationRequest:
		resp := a.handleActivationRequest(msg)
		c.Respond(resp)
	case getKinds:
		kinds := make([]string, len(a.kinds))
		i := 0
		for kind := range a.kinds {
			kinds[i] = kind
			i++
		}
		c.Respond(kinds)
	}
}

func (a *Agent) handleActorTopology(msg *ActorTopology) {
	for _, actorInfo := range msg.Actors {
		a.addActiveKind(actorInfo.CID)
	}
}

// A new kind is activated on this cluster.
func (a *Agent) handleActivation(msg *Activation) {
	a.addActiveKind(msg.CID)
}

func (a *Agent) handleActivationRequest(msg *ActivationRequest) *ActivationResponse {
	if !a.hasKindLocal(msg.Kind) {
		slog.Error("received activation request but kind not registered locally on this node", "kind", msg.Kind)
		return &ActivationResponse{Success: false}
	}
	kind := a.localKinds[msg.Kind]
	// Spawn the actor (receiver) with kind/id
	pid := a.cluster.engine.Spawn(kind.producer, msg.Kind+"/"+msg.ID)
	resp := &ActivationResponse{
		CID:     NewCID(pid, msg.Kind, msg.ID, a.cluster.region),
		Success: true,
	}
	return resp
}

func (a *Agent) activate(kind, id string) *CID {
	var (
		// TODO: pick member based on rendezvous and custom strategy
		members      = a.members.FilterByKind(kind)
		owner        = members[rand.Intn(len(members))]
		activatorPID = actor.NewPID(owner.Host, "cluster/"+owner.ID)
		req          = &ActivationRequest{Kind: kind, ID: id}
	)

	var activationResp *ActivationResponse
	// Local activation
	if owner.Host == a.cluster.engine.Address() {
		activationResp = a.handleActivationRequest(req)
	} else {
		// Remote activation
		// TODO: topology hash
		// TODO: retry this couple times
		resp, err := a.cluster.engine.Request(activatorPID, req, time.Millisecond*100).Result()
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
			slog.Error("activation unsuccessfull", "msg", r)
			return nil
		}
		activationResp = r
	}

	a.bcast(&Activation{
		CID: activationResp.CID,
	})

	return activationResp.CID
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

	actorInfos := []*ActorInfo{}
	for _, kinds := range a.activeKinds.kinds {
		for _, akind := range kinds.ToSlice() {
			actorInfo := &ActorInfo{
				CID: akind.cid,
			}
			actorInfos = append(actorInfos, actorInfo)
		}
	}

	// Send our ActorTopology to this member
	if len(actorInfos) > 0 {
		a.cluster.engine.Send(member.PID(), &ActorTopology{Actors: actorInfos})
	}

	slog.Info("member joined", "we", a.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
}

func (a *Agent) memberLeave(member *Member) {
	slog.Info("member left", "we", a.cluster.id, "id", member.ID, "host", member.Host, "kinds", member.Kinds)
	a.members.Remove(member)
}

func (a *Agent) bcast(msg any) {
	a.members.ForEach(func(member *Member) bool {
		a.cluster.engine.Send(member.PID(), msg)
		return true
	})
}

func (a *Agent) addActiveKind(cid *CID) {
	akind := ActiveKind{
		cid:     cid,
		isLocal: false,
	}
	if !a.activeKinds.Has(akind) {
		a.activeKinds.Add(akind)
		slog.Info("[ACTIVE]", "we", a.cluster.id, "cid", cid)
	}
}

func (a *Agent) hasKindLocal(name string) bool {
	for _, kind := range a.localKinds {
		if kind.name == name {
			return true
		}
	}
	return false
}
