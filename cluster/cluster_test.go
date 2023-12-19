package cluster

import (
	"log"
	"sync"
	"testing"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
	"github.com/stretchr/testify/assert"
)

type Player struct{}

func NewPlayer() actor.Receiver {
	return &Player{}
}

func (p Player) Receive(c *actor.Context) {}

type Inventory struct{}

func NewInventory() actor.Receiver {
	return &Inventory{}
}

func (i Inventory) Receive(c *actor.Context) {}

func TestRegisterKind(t *testing.T) {
	c := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c.RegisterKind("player", NewPlayer, KindConfig{})
	c.RegisterKind("inventory", NewInventory, KindConfig{})
	assert.True(t, c.HasKindLocal("player"))
	assert.True(t, c.HasKindLocal("inventory"))
}

func TestMemberJoin(t *testing.T) {
	c1 := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c2 := makeCluster(t, "127.0.0.1:3001", "B", "eu-west", MemberAddr{
		ListenAddr: "127.0.0.1:3000",
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, KindConfig{})
	c1.Start()

	wg := sync.WaitGroup{}
	wg.Add(2)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		// we do this so we are 100% sure nodes are connected with eachother.
		case MemberJoinEvent:
			_ = msg
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)
	c2.Start()

	wg.Wait()
	agent := c1.engine.Registry.GetReceiver("cluster/A").(*Agent)
	assert.Equal(t, agent.members.Len(), 2)
	assert.True(t, agent.kinds["player"])
}

func TestActivate(t *testing.T) {
	c1 := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c2 := makeCluster(t, "127.0.0.1:3001", "B", "eu-west", MemberAddr{
		ListenAddr: "127.0.0.1:3000",
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, KindConfig{})
	c1.Start()

	expectedPID := actor.NewPID(c2.engine.Address(), "player/1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		// we do this so we are 100% sure nodes are connected with eachother.
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				// Because c1 doesnt have player registered locally we can only spawned
				// the player on c2
				pid := c1.Activate("player", ActivationConfig{ID: "1"})
				assert.True(t, pid.Equals(expectedPID))
			}
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)

	c2.Start()

	wg.Wait()
	agent := c1.engine.Registry.GetReceiver("cluster/A").(*Agent)
	assert.Equal(t, agent.members.Len(), 2)
	assert.True(t, agent.kinds["player"])
	assert.True(t, agent.activated["player/1"].Equals(expectedPID))
}

func TestDeactivate(t *testing.T) {
	c1 := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c2 := makeCluster(t, "127.0.0.1:3001", "B", "eu-west", MemberAddr{
		ListenAddr: "127.0.0.1:3000",
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, KindConfig{})
	c1.Start()

	expectedPID := actor.NewPID(c2.engine.Address(), "player/1")
	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				pid := c1.Activate("player", ActivationConfig{ID: "1"})
				assert.True(t, pid.Equals(expectedPID))
			}
		case ActivationEvent:
			c1.Deactivate(msg.PID)
		case DeactivationEvent:
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)

	c2.Start()
	wg.Wait()

	agent := c1.engine.Registry.GetReceiver("cluster/A").(*Agent)
	assert.Equal(t, agent.members.Len(), 2)
	assert.True(t, agent.kinds["player"])
	assert.Nil(t, agent.activated["player/1"])
}

func TestMemberLeave(t *testing.T) {
	memberLeaveAddr := "127.0.0.1:3001"
	remote := remote.New(remote.Config{
		ListenAddr: memberLeaveAddr,
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	cfg := Config{
		ClusterProvider: NewSelfManagedProvider(MemberAddr{
			ListenAddr: "127.0.0.1:3000",
			ID:         "A",
		}),
		ID:     "B",
		Region: "eu-east",
		Engine: e,
	}
	c2, err := New(cfg)
	assert.Nil(t, err)

	c1 := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c2.RegisterKind("player", NewPlayer, KindConfig{})
	c1.Start()

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				remote.Stop()
			}
		case MemberLeaveEvent:
			assert.Equal(t, msg.Member.ID, c2.id)
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)
	c2.Start()

	wg.Wait()
	agent := c1.engine.Registry.GetReceiver("cluster/A").(*Agent)
	assert.Equal(t, agent.members.Len(), 1)
	assert.False(t, agent.kinds["player"])
}

func TestMembersExcept(t *testing.T) {
	a := []*Member{
		{
			ID:   "A",
			Host: ":3000",
		},
		{
			ID:   "B",
			Host: ":3001",
		},
	}
	b := []*Member{
		{
			ID:   "A",
			Host: ":3000",
		},
		{
			ID:   "B",
			Host: ":3001",
		},
		{
			ID:   "C",
			Host: ":3002",
		},
	}
	am := NewMemberSet(b...).Except(a)
	assert.Len(t, am, 1)
	assert.Equal(t, am[0].ID, "C")
}

func makeCluster(t *testing.T, addr, id, region string, members ...MemberAddr) *Cluster {
	remote := remote.New(remote.Config{
		ListenAddr: addr,
	})
	e, err := actor.NewEngine(actor.EngineOptRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	cfg := Config{
		ClusterProvider: NewSelfManagedProvider(members...),
		ID:              id,
		Region:          region,
		Engine:          e,
	}
	c, err := New(cfg)
	assert.Nil(t, err)
	return c
}
