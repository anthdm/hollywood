package cluster

import (
	"fmt"
	"log"
	"math/rand"
	sync "sync"
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
	c := makeCluster(t, getRandomLocalhostAddr(), "A", "eu-west")
	c.RegisterKind("player", NewPlayer, nil)
	c.RegisterKind("inventory", NewInventory, nil)
	assert.True(t, c.HasKindLocal("player"))
	assert.True(t, c.HasKindLocal("inventory"))
}

func TestClusterSpawn(t *testing.T) {
	c1Addr := getRandomLocalhostAddr()
	c1 := makeCluster(t, c1Addr, "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west", MemberAddr{
		ListenAddr: c1Addr,
		ID:         "A",
	})

	expectedPID := actor.NewPID(c1Addr, "player/1")

	wg := sync.WaitGroup{}
	wg.Add(2)

	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				c1.Spawn(NewPlayer, "player", actor.WithID("1"))
			}
		case ActivationEvent:
			assert.True(t, msg.PID.Equals(expectedPID))
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)

	eventPIDc2 := c2.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case ActivationEvent:
			assert.True(t, msg.PID.Equals(expectedPID))
			wg.Done()
		}
	}, "event")
	c2.engine.Subscribe(eventPIDc2)

	c1.Start()
	c2.Start()
	wg.Wait()
}

func TestMemberJoin(t *testing.T) {
	addr := getRandomLocalhostAddr()
	c1 := makeCluster(t, addr, "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west", MemberAddr{
		ListenAddr: addr,
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, nil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		// we do this so we are 100% sure nodes are connected with eachother.
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				_ = msg
				wg.Done()
			}
		}
	}, "event")
	c1.engine.Subscribe(eventPID)
	c1.Start()
	c2.Start()

	wg.Wait()
	assert.Equal(t, len(c1.Members()), 2)
	assert.True(t, c1.HasKind("player"))
}

func TestActivate(t *testing.T) {
	addr := getRandomLocalhostAddr()
	c1 := makeCluster(t, addr, "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west", MemberAddr{
		ListenAddr: addr,
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, nil)

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
				pid := c1.Activate("player", &ActivationConfig{ID: "1"})
				assert.True(t, pid.Equals(expectedPID))
			}
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)

	c1.Start()
	c2.Start()

	wg.Wait()
	assert.Equal(t, len(c1.Members()), 2)
	assert.True(t, c1.HasKind("player"))
	assert.True(t, c1.GetActivated("player/1").Equals(expectedPID))
}

func TestDeactivate(t *testing.T) {
	addr := getRandomLocalhostAddr()
	c1 := makeCluster(t, addr, "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west", MemberAddr{
		ListenAddr: addr,
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, nil)

	expectedPID := actor.NewPID(c2.engine.Address(), "player/1")
	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				pid := c1.Activate("player", &ActivationConfig{ID: "1"})
				assert.True(t, pid.Equals(expectedPID))
			}
		case ActivationEvent:
			c1.Deactivate(msg.PID)
		case DeactivationEvent:
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)

	c1.Start()
	c2.Start()
	wg.Wait()

	assert.Equal(t, len(c1.Members()), 2)
	assert.True(t, c1.HasKind("player"))
	assert.Nil(t, c1.GetActivated("player/1"))
}

func TestMemberLeave(t *testing.T) {
	c1Addr := getRandomLocalhostAddr()
	c2Addr := getRandomLocalhostAddr()
	remote := remote.New(c2Addr, nil)

	e, err := actor.NewEngine(&actor.EngineOpts{Remote: remote})
	if err != nil {
		log.Fatal(err)
	}
	cfg := Config{
		ClusterProvider: NewSelfManagedProvider(MemberAddr{
			ListenAddr: c1Addr,
			ID:         "A",
		}),
		ID:     "B",
		Region: "eu-east",
		Engine: e,
	}
	c2, err := New(cfg)
	assert.Nil(t, err)

	c1 := makeCluster(t, c1Addr, "A", "eu-west")
	c2.RegisterKind("player", NewPlayer, nil)
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
	assert.Equal(t, len(c1.Members()), 1)
	assert.False(t, c1.HasKind("player"))
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
	remote := remote.New(addr, nil)
	e, err := actor.NewEngine(&actor.EngineOpts{Remote: remote})
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

func getRandomLocalhostAddr() string {
	return fmt.Sprintf("localhost:%d", rand.Intn(50000)+10000)
}
