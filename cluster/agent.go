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

// activate is a message type used to request the activation of an entity in the cluster.
type activate struct {
	kind   string // kind represents the type or category of the entity to be activated.
	id     string // id is the unique identifier of the entity to be activated.
	region string // region specifies the cluster region where the entity should be activated.
}

// deactivate is a message type used to request the deactivation of an entity in the cluster.
type deactivate struct {
	pid *actor.PID // pid is the actor's process identifier that should be deactivated.
}

// Agent is a struct that represents an agent within a cluster.
// It manages cluster members, kinds, and the activation of entities.
type Agent struct {
	members *MemberSet // members is a set that keeps track of the members of the cluster.
	cluster *Cluster   // cluster is a reference to the cluster this agent is part of.

	kinds map[string]bool // kinds is a map tracking the types of entities managed by the agent.

	localKinds map[string]kind // localKinds is a map of entity kinds that are local to the agent.

	// activated is a map that tracks all the actors that are active across the cluster.
	activated map[string]*actor.PID
}

// NewAgent is a function that creates and returns a new agent for the cluster.
// It initializes the agent with the kinds of entities it manages and returns a Producer that creates a Receiver.
func NewAgent(c *Cluster) actor.Producer {

	kinds := make(map[string]bool)
	localKinds := make(map[string]kind)

	// Populate the kinds and localKinds maps based on the kinds available in the cluster.
	for _, kind := range c.kinds {
		kinds[kind.name] = true      // Set the kind name as a key in the kinds map.
		localKinds[kind.name] = kind // Associate the kind with its name in the localKinds map.
	}

	// Return a Producer function that, when called, creates and returns a new Agent as a Receiver.
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

// Receive is a method of the Agent type that handles incoming messages.
// It acts upon different types of messages based on their type.
func (a *Agent) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		// Handle the Started message (no action required).

	case *ActorTopology:
		// Handle an ActorTopology message.
		a.handleActorTopology(msg)

	case *Members:
		// Handle a Members message which contains information about cluster members.
		a.handleMembers(msg.Members)

	case *Activation:
		// Handle an Activation message to activate an entity.
		a.handleActivation(msg)

	case activate:
		// Handle an activate message, which requests activation of a specific entity.
		pid := a.activate(msg.kind, msg.id, msg.region)
		c.Respond(pid) // Respond with the PID of the activated entity.

	case deactivate:
		// Handle a deactivate message by broadcasting a Deactivation message.
		a.bcast(&Deactivation{PID: msg.pid})

	case *Deactivation:
		// Handle a Deactivation message to deactivate an entity.
		a.handleDeactivation(msg)

	case *ActivationRequest:
		resp := a.handleActivationRequest(msg)
		c.Respond(resp)

	case getMembers:
		c.Respond(a.members.Slice())

	case getKinds:
		// Handle a getKinds message by responding with a slice of kinds.
		kinds := make([]string, len(a.kinds))
		i := 0
		for kind := range a.kinds {
			kinds[i] = kind
			i++
		}
		c.Respond(kinds)

	case getActive:
		// Handle a getActive message by responding with the PID of the requested active entity.
		pid := a.activated[msg.id]
		c.Respond(pid)
	}
}

// handleActorTopology processes an ActorTopology message.
// It updates the agent's state based on the topology information received.
func (a *Agent) handleActorTopology(msg *ActorTopology) {
	for _, actorInfo := range msg.Actors {
		// Add each actor's PID to the activated list of the agent.
		a.addActivated(actorInfo.PID)
	}
}

// handleDeactivation processes a Deactivation message.
// It removes the deactivated actor from the agent's state and notifies the cluster of the deactivation.
func (a *Agent) handleDeactivation(msg *Deactivation) {
	// Remove the actor's PID from the activated list.
	a.removeActivated(msg.PID)

	// Send a Poison message to the actor's PID to shut it down.
	a.cluster.engine.Poison(msg.PID)

	// Broadcast a DeactivationEvent to the cluster, notifying other components of the deactivation.
	a.cluster.engine.BroadcastEvent(DeactivationEvent{PID: msg.PID})
}

// handleActivation processes an Activation message, indicating a new kind has been activated on the cluster.
func (a *Agent) handleActivation(msg *Activation) {
	// Add the activated actor's PID to the agent's list of activated actors.
	a.addActivated(msg.PID)

	// Broadcast an ActivationEvent to the cluster to notify other components of the new activation.
	a.cluster.engine.BroadcastEvent(ActivationEvent{PID: msg.PID})
}

// handleActivationRequest processes an ActivationRequest message.
// It checks if the requested kind is available locally and, if so, activates it.
func (a *Agent) handleActivationRequest(msg *ActivationRequest) *ActivationResponse {

	// Check if the requested kind is registered locally on this node.
	if !a.hasKindLocal(msg.Kind) {
		slog.Error("received activation request but kind not registered locally on this node", "kind", msg.Kind)
		return &ActivationResponse{Success: false}
	}

	// Retrieve the kind details from the local kinds map.
	kind := a.localKinds[msg.Kind]

	// Spawn a new actor of the requested kind and with the specified ID.
	pid := a.cluster.engine.Spawn(kind.producer, msg.Kind, actor.WithID(msg.ID))

	// Create a response indicating success and include the PID of the newly spawned actor.
	resp := &ActivationResponse{
		PID:     pid,
		Success: true,
	}
	return resp
}

// activate attempts to activate an actor of a specified kind and ID within a given region.
// It returns the PID of the activated actor or nil if activation fails.
func (a *Agent) activate(kind, id, region string) *actor.PID {
	// Filter the members of the cluster by the specified kind.
	members := a.members.FilterByKind(kind)
	if len(members) == 0 {
		slog.Warn("could not find any members with kind", "kind", kind)
		return nil
	}

	// Determine the member to activate the actor on, based on the activation strategy.
	owner := a.cluster.activationStrategy.ActivateOnMember(ActivationDetails{
		Members: members,
		Region:  region,
		Kind:    kind,
	})
	if owner == nil {
		slog.Warn("activator did not found a member to activate on")
		return nil
	}

	// Create an ActivationRequest for the actor.
	req := &ActivationRequest{Kind: kind, ID: id}
	// Create a PID for the activator based on the owner's host and ID.
	activatorPID := actor.NewPID(owner.Host, "cluster/"+owner.ID)

	var activationResp *ActivationResponse

	// Local activation if the owner's host matches the current engine's address.
	if owner.Host == a.cluster.engine.Address() {
		activationResp = a.handleActivationRequest(req)
	} else {
		// Remote activation if the owner is on a different host.
		// @TODO: topology hash
		resp, err := a.cluster.engine.Request(activatorPID, req, requestTimeout).Result()
		if err != nil {
			// Log an error if the activation request fails.
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

	// Broadcast an Activation message to the cluster after successful activation.
	a.bcast(&Activation{
		PID: activationResp.PID,
	})

	return activationResp.PID
}

// handleMembers processes a list of cluster members, determining which members have joined or left.
func (a *Agent) handleMembers(members []*Member) {
	// Create a set of new members and determine which members have joined since the last update.
	// This is done by excluding existing members from the new list.
	joined := NewMemberSet(members...).Except(a.members.Slice())

	// Determine which members have left by excluding the new list from existing members.
	left := a.members.Except(members)

	for _, member := range joined {
		a.memberJoin(member)
	}

	for _, member := range left {
		a.memberLeave(member)
	}
}

// memberJoin handles the actions to be taken when a new member joins the cluster.
func (a *Agent) memberJoin(member *Member) {
	// Add the new member to the agent's member set.
	a.members.Add(member)

	// Update the kinds available in the cluster based on the new member's kinds.
	// This keeps track of all kinds available across the cluster.
	for _, kind := range member.Kinds {
		if _, ok := a.kinds[kind]; !ok {
			a.kinds[kind] = true
		}
	}

	// Prepare a list of ActorInfo for all activated actors.
	actorInfos := make([]*ActorInfo, 0)
	for _, pid := range a.activated {
		actorInfo := &ActorInfo{
			PID: pid,
		}
		actorInfos = append(actorInfos, actorInfo)
	}

	// If there are any activated actors, send the ActorTopology to the new member.
	if len(actorInfos) > 0 {
		a.cluster.engine.Send(member.PID(), &ActorTopology{Actors: actorInfos})
	}

	// Broadcast a MemberJoinEvent to the cluster, notifying other components of the new member's arrival.
	a.cluster.engine.BroadcastEvent(MemberJoinEvent{
		Member: member,
	})

	slog.Debug("member joined", "id", member.ID, "host", member.Host, "kinds", member.Kinds, "region", member.Region)
}

// memberLeave handles the actions to be taken when a member leaves the cluster.
func (a *Agent) memberLeave(member *Member) {
	// Remove the member from the agent's member set.
	a.members.Remove(member)

	// Rebuild the kinds list to reflect the current state of the cluster after the member's departure.
	a.rebuildKinds()

	// Iterate through all activated actors and deactivate those that were running on the member that left.
	for _, pid := range a.activated {
		if pid.Address == member.Host {
			// If the actor was running on the member that left, deactivate it.
			a.removeActivated(pid)
		}
	}

	// Broadcast a MemberLeaveEvent to the cluster, notifying other components of the member's departure.
	a.cluster.engine.BroadcastEvent(MemberLeaveEvent{Member: member})

	slog.Debug("member left", "id", member.ID, "host", member.Host, "kinds", member.Kinds)
}

// bcast broadcasts a message to all members of the cluster.
func (a *Agent) bcast(msg any) {
	a.members.ForEach(func(member *Member) bool {
		a.cluster.engine.Send(member.PID(), msg)
		return true
	})
}

// addActivated adds an actor's PID to the list of activated actors if it's not already present.
func (a *Agent) addActivated(pid *actor.PID) {
	// Check if the actor's PID is already in the activated list.
	if _, ok := a.activated[pid.ID]; !ok {
		// If not, add the PID to the activated list.
		a.activated[pid.ID] = pid
		slog.Debug("new actor available on cluster", "pid", pid)
	}
}

// removeActivated removes an actor's PID from the list of activated actors.
func (a *Agent) removeActivated(pid *actor.PID) {
	// Delete the actor's PID from the activated list.
	delete(a.activated, pid.ID)
	slog.Debug("actor removed from cluster", "pid", pid)
}

// hasKindLocal checks if a given kind name is registered locally on this node.
func (a *Agent) hasKindLocal(name string) bool {
	_, ok := a.localKinds[name]
	return ok // Return true if the kind is registered locally, false otherwise.
}

// rebuildKinds updates the kinds map to reflect the current kinds available in the cluster.
func (a *Agent) rebuildKinds() {
	// Clear the existing kinds map to prepare for rebuilding.
	maps.Clear(a.kinds)

	a.members.ForEach(func(m *Member) bool {
		for _, kind := range m.Kinds {
			// Add each kind to the kinds map if it's not already present.
			if _, ok := a.kinds[kind]; !ok {
				a.kinds[kind] = true
			}
		}
		return true
	})
}
