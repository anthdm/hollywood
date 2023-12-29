package cluster

import (
	"time"

	"github.com/anthdm/hollywood/actor"
)

// memberPingInterval defines the interval at which member ping messages are sent.
// It is set to 5 seconds.
const memberPingInterval = time.Second * 5

// MemberAddr represents the address and ID of a cluster member.
type MemberAddr struct {
	ListenAddr string // ListenAddr is the listening address of the member.
	ID         string // ID is the unique identifier of the member.
}

// memberLeave is a message type used to signal the departure of a member from the cluster.
type memberLeave struct {
	ListenAddr string // ListenAddr is the address of the member that is leaving.
}

// memberPing is a message type used for pinging members to check their availability.
type memberPing struct{}

// SelfManaged is a type that manages a cluster in a self-contained manner.
// It includes functionality for managing cluster members and handling cluster events.
type SelfManaged struct {
	cluster        *Cluster           // cluster is a reference to the Cluster that this instance is managing.
	bootstrapAddrs []MemberAddr       // bootstrapAddrs is a slice of MemberAddr used for initializing the cluster.
	members        *MemberSet         // members is the set of current members in the cluster.
	memberPinger   actor.SendRepeater // memberPinger is used to periodically send ping messages to cluster members.
	eventSubPID    *actor.PID         // eventSubPID is the PID for subscribing to cluster events.

	pid *actor.PID // pid is the process identifier for the SelfManaged instance.

	membersAlive *MemberSet // membersAlive is the set of members that are currently alive and responsive.
}

// NewSelfManagedProvider creates a Producer that returns a new SelfManaged instance.
// This function is used for initializing a SelfManaged provider with given bootstrap addresses.
func NewSelfManagedProvider(addrs ...MemberAddr) Producer {
	// The function returns a Producer, which itself returns an actor.Producer.
	return func(c *Cluster) actor.Producer {
		// The returned actor.Producer, when invoked, creates and returns a SelfManaged actor.Receiver.
		return func() actor.Receiver {
			// Initialize and return a SelfManaged instance with the provided cluster and bootstrap addresses.
			return &SelfManaged{
				cluster:        c,              // Set the cluster reference.
				bootstrapAddrs: addrs,          // Set the bootstrap addresses.
				members:        NewMemberSet(), // Initialize an empty MemberSet for current members.
				membersAlive:   NewMemberSet(), // Initialize an empty MemberSet for alive members.
			}
		}
	}
}

// Receive is a method of the SelfManaged type that processes incoming messages.
// It acts upon different types of messages based on their type.
func (s *SelfManaged) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		// Handle the Started message: set up the SelfManaged instance when it starts.
		s.pid = c.PID()                   // Store the PID of the SelfManaged instance.
		s.members.Add(s.cluster.Member()) // Add the current cluster member to the members set.
		// Create a Members message with the current members and send it to the cluster's PID.
		members := &Members{
			Members: s.members.Slice(),
		}
		s.cluster.engine.Send(s.cluster.PID(), members)
		// Start a repeater to send memberPing messages at regular intervals.
		s.memberPinger = c.SendRepeat(c.PID(), memberPing{}, memberPingInterval)
		// Start additional setup procedures.
		s.start(c)

	case actor.Stopped:
		// Handle the Stopped message: perform cleanup when the SelfManaged instance stops.
		s.memberPinger.Stop()                       // Stop the member pinger.
		s.cluster.engine.Unsubscribe(s.eventSubPID) // Unsubscribe from cluster events.

	case *MembersJoin:
		// Handle MembersJoin messages: add new members to the cluster.
		for _, member := range msg.Members {
			s.addMember(member) // Add each new member to the SelfManaged instance's members set.
		}
		// Send an updated Members message to all members in the cluster.
		ourMembers := &Members{
			Members: s.members.Slice(),
		}
		s.members.ForEach(func(member *Member) bool {
			s.cluster.engine.Send(memberToProviderPID(member), ourMembers)
			return true
		})

	case *Members:
		// Handle Members messages: update the list of members.
		for _, member := range msg.Members {
			s.addMember(member) // Add each member to the SelfManaged instance's members set.
		}
		// If there are members, send an updated Members message to the cluster's PID.
		if s.members.Len() > 0 {
			members := &Members{
				Members: s.members.Slice(),
			}
			s.cluster.engine.Send(s.cluster.PID(), members)
		}

	case memberPing:
		// Handle memberPing messages: send ping messages to all members except self.
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
		// Handle memberLeave messages: remove a member that has left the cluster.
		member := s.members.GetByHost(msg.ListenAddr)
		s.removeMember(member) // Remove the member from the SelfManaged instance's members set.
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

// removeMember handles the removal of a member from the cluster.
// It updates the cluster state after the member has been removed.
func (s *SelfManaged) removeMember(member *Member) {
	// Check if the member is in the SelfManaged instance's member set.
	if s.members.Contains(member) {
		// If the member is found, remove it from the member set.
		s.members.Remove(member)
	}

	// Update the cluster state to reflect the change in membership.
	s.updateCluster()
}

// updateCluster sends an updated list of members to the cluster.
// This method is used to inform the cluster about the current state of its members.
func (s *SelfManaged) updateCluster() {
	// Create a Members message containing a slice of the current members.
	members := &Members{
		Members: s.members.Slice(),
	}

	// Send the Members message to the cluster's PID.
	// This action updates the cluster with the latest information about its members.
	s.cluster.engine.Send(s.cluster.PID(), members)
}

// start initializes the SelfManaged instance with necessary setup procedures.
func (s *SelfManaged) start(c *actor.Context) {
	// Spawn a child actor for handling specific events.
	s.eventSubPID = c.SpawnChildFunc(func(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case actor.RemoteUnreachableEvent:
			// Handle RemoteUnreachableEvent by sending a memberLeave message to self.
			// This indicates that a remote member is unreachable and should be considered as having left the cluster.
			ctx.Send(s.pid, memberLeave{ListenAddr: msg.ListenAddr})
		}
	}, "event") // The child actor is named "event".

	// Subscribe the event child actor to cluster events.
	s.cluster.engine.Subscribe(s.eventSubPID)

	// Create a MembersJoin message with the current members.
	members := &MembersJoin{
		Members: s.members.Slice(),
	}

	// Iterate over the bootstrap addresses and send the MembersJoin message to each member.
	for _, ma := range s.bootstrapAddrs {
		// Create a PID for the member at the bootstrap address.
		memberPID := actor.NewPID(ma.ListenAddr, "provider/"+ma.ID)
		// Send the MembersJoin message to the member PID.
		s.cluster.engine.Send(memberPID, members)
	}
}
