package cluster

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/grandcat/zeroconf"
)

const (
	serviceName        = "_actor.hollywood_"
	domain             = "local."
	memberPingInterval = time.Second * 2
)

// MemberAddr represents a reachable node in the cluster.
type MemberAddr struct {
	ListenAddr string
	ID         string
}

type (
	memberLeave struct {
		ListenAddr string
	}
	memberPing struct{}
)

type SelfManagedConfig struct {
	bootstrapMembers []MemberAddr
}

func NewSelfManagedConfig() SelfManagedConfig {
	return SelfManagedConfig{
		bootstrapMembers: make([]MemberAddr, 0),
	}
}

func (c SelfManagedConfig) WithBootstrapMember(member MemberAddr) SelfManagedConfig {
	c.bootstrapMembers = append(c.bootstrapMembers, member)
	return c
}

type SelfManaged struct {
	config       SelfManagedConfig
	cluster      *Cluster
	members      *MemberSet
	memberPinger actor.SendRepeater
	eventSubPID  *actor.PID

	pid *actor.PID

	membersAlive *MemberSet

	resolver  *zeroconf.Resolver
	announcer *zeroconf.Server

	ctx    context.Context
	cancel context.CancelFunc
}

func NewSelfManagedProvider(config SelfManagedConfig) Producer {
	return func(c *Cluster) actor.Producer {
		return func() actor.Receiver {
			return &SelfManaged{
				config:       config,
				cluster:      c,
				members:      NewMemberSet(),
				membersAlive: NewMemberSet(),
			}
		}
	}
}

func (s *SelfManaged) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.ctx, s.cancel = context.WithCancel(context.Background())
		s.pid = c.PID()

		s.members.Add(s.cluster.Member())
		s.sendMembersToAgent()

		s.memberPinger = c.SendRepeat(c.PID(), memberPing{}, memberPingInterval)
		s.start(c)
	case actor.Stopped:
		s.memberPinger.Stop()
		s.cluster.engine.Unsubscribe(s.eventSubPID)
		s.announcer.Shutdown()
		s.cancel()
	case *Handshake:
		s.addMembers(msg.Member)
		members := s.members.Slice()
		s.cluster.engine.Send(c.Sender(), &Members{
			Members: members,
		})
	case *Members:
		s.addMembers(msg.Members...)
	case memberPing:
		s.handleMemberPing(c)
	case memberLeave:
		member := s.members.GetByHost(msg.ListenAddr)
		s.removeMember(member)
	case *actor.Ping:
	case actor.Initialized:
		_ = msg
	default:
		slog.Warn("received unhandled message", "msg", msg, "t", reflect.TypeOf(msg))
	}
}

func (s *SelfManaged) handleMemberPing(c *actor.Context) {
	s.members.ForEach(func(member *Member) bool {
		if member.Host != s.cluster.agentPID.Address {
			ping := &actor.Ping{
				From: c.PID(),
			}
			c.Send(memberToProviderPID(member), ping)
		}
		return true
	})
}

func (s *SelfManaged) addMembers(members ...*Member) {
	for _, member := range members {
		if !s.members.Contains(member) {
			s.members.Add(member)
		}
	}
	s.sendMembersToAgent()
}

func (s *SelfManaged) removeMember(member *Member) {
	if s.members.Contains(member) {
		s.members.Remove(member)
	}
	s.sendMembersToAgent()
}

// send all the current members to the local cluster agent.
func (s *SelfManaged) sendMembersToAgent() {
	members := &Members{
		Members: s.members.Slice(),
	}
	s.cluster.engine.Send(s.cluster.PID(), members)
}

func (s *SelfManaged) start(c *actor.Context) {
	s.eventSubPID = c.SpawnChildFunc(s.handleEventStream, "event")
	s.cluster.engine.Subscribe(s.eventSubPID)

	// send handshake to all bootstrap members if any.
	for _, member := range s.config.bootstrapMembers {
		memberPID := actor.NewPID(member.ListenAddr, "provider/"+member.ID)
		s.cluster.engine.SendWithSender(memberPID, &Handshake{
			Member: s.cluster.Member(),
		}, c.PID())
	}

	s.initAutoDiscovery()
	s.startAutoDiscovery()
}

func (s *SelfManaged) initAutoDiscovery() {
	resolver, err := zeroconf.NewResolver()
	if err != nil {
		log.Fatal(err)
	}
	s.resolver = resolver

	host, portstr, err := net.SplitHostPort(s.cluster.agentPID.Address)
	if err != nil {
		log.Fatal(err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := zeroconf.RegisterProxy(
		s.cluster.ID(),
		serviceName,
		domain,
		port,
		fmt.Sprintf("member_%s", s.cluster.ID()),
		[]string{host},
		[]string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		log.Fatal(err)
	}
	s.announcer = server
}

func (s *SelfManaged) startAutoDiscovery() {
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if entry.Instance != s.cluster.ID() {
				host := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
				hs := &Handshake{
					Member: s.cluster.Member(),
				}
				// create the reachable PID for this member.
				memberPID := actor.NewPID(host, "provider/"+entry.Instance)
				self := actor.NewPID(s.cluster.agentPID.Address, "provider/"+s.cluster.ID())
				s.cluster.engine.SendWithSender(memberPID, hs, self)
			}
		}
		slog.Info("[CLUSTER] stopping discovery", "id", s.cluster.ID())
	}(entries)

	err := s.resolver.Browse(s.ctx, serviceName, domain, entries)
	if err != nil {
		slog.Error("[CLUSTER] discovery failed", "err", err)
		panic(err)
	}
}

func (s *SelfManaged) handleEventStream(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.RemoteUnreachableEvent:
		c.Send(s.pid, memberLeave{ListenAddr: msg.ListenAddr})
	}
}
