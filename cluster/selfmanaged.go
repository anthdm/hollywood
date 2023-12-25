package cluster

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/grandcat/zeroconf"
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

	resolver  *zeroconf.Resolver
	announcer *zeroconf.Server
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
		go func() {
			for {
				time.Sleep(time.Second * 5)
				fmt.Println(s.members.Slice())
			}
		}()
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
	case *Handshake:
		s.addMember(msg.Member)
		s.cluster.engine.Send(c.Sender(), s.cluster.Member())
	case *Member:
		s.addMember(msg)
	case *Members:
		s.handleMembers(msg.Members)
	case memberPing:
		s.handleMemberPing(c)
	case memberLeave:
		member := s.members.GetByHost(msg.ListenAddr)
		s.removeMember(member)
	}
}

func (s *SelfManaged) handleMembers(members []*Member) {
	for _, member := range members {
		s.addMember(member)
	}
	if s.members.Len() > 0 {
		members := &Members{
			Members: s.members.Slice(),
		}
		s.cluster.engine.Send(s.cluster.PID(), members)
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

func (s *SelfManaged) addMember(member *Member) {
	if !s.members.Contains(member) {
		s.members.Add(member)
	}
	s.updateCluster()
}

func (s *SelfManaged) removeMember(member *Member) {
	if s.members.Contains(member) {
		s.members.Remove(member)
	}
	s.updateCluster()
}

// updates the local member of the cluster.
func (s *SelfManaged) updateCluster() {
	members := &Members{
		Members: s.members.Slice(),
	}
	s.cluster.engine.Send(s.cluster.PID(), members)
}

func (s *SelfManaged) start(c *actor.Context) {
	s.eventSubPID = c.SpawnChildFunc(s.handleEventStream, "event")
	s.cluster.engine.Subscribe(s.eventSubPID)

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
		s.cluster.id,
		"_hollywood_",
		"local.",
		port,
		"member1",
		[]string{host},
		[]string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}
	s.announcer = server

	s.startDiscovery()
}

func (s *SelfManaged) startDiscovery() {
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if entry.Instance != s.cluster.id {
				host := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
				hs := &Handshake{
					Member: s.cluster.Member(),
				}
				// create the reachable PID for this member.
				memberPID := actor.NewPID(host, "provider/"+entry.Instance)
				self := actor.NewPID(s.cluster.agentPID.Address, "provider/"+s.cluster.id)
				s.cluster.engine.SendWithSender(memberPID, hs, self)
			}
		}
	}(entries)

	ctx := context.Background()
	err := s.resolver.Browse(ctx, "_hollywood_", "local.", entries)
	if err != nil {
		slog.Error("[DISCOVERY] starting discovery failed", "err", err)
		panic(err)
	}
}

func (s *SelfManaged) handleEventStream(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.RemoteUnreachableEvent:
		c.Send(s.pid, memberLeave{ListenAddr: msg.ListenAddr})
	}
}
