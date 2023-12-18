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

func TestActivate(t *testing.T) {
	c1 := makeCluster(t, "127.0.0.1:3000", "A", "eu-west")
	c2 := makeCluster(t, "127.0.0.1:3001", "B", "eu-west", MemberAddr{
		ListenAddr: "127.0.0.1:3000",
		ID:         "A",
	})
	c2.RegisterKind("player", NewPlayer, KindConfig{})
	c1.Start()
	c2.Start()

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		// we do this so we are 100% sure nodes are connected with eachother.
		case MemberJoinEvent:
			pid := c1.Activate("player", ActivationConfig{ID: "1"})
			// Because c1 doesnt have player registered locally we can only spawned
			// the player on c2
			expectedPID := actor.NewPID(c2.engine.Address(), "player/1")
			assert.True(t, pid.Equals(expectedPID))
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)
	wg.Wait()
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
