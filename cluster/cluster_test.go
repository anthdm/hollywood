package cluster

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestClusterActivationOnMemberFunc(t *testing.T) {
	c, err := New(NewConfig())
	require.Nil(t, err)

	c.RegisterKind("player", NewPlayer, NewKindConfig())
}

func TestClusterShouldWorkWithDefaultValues(t *testing.T) {
	config := NewConfig()
	c, err := New(config)
	assert.Nil(t, err)
	assert.True(t, len(c.config.id) > 0)
	assert.Equal(t, c.config.region, "default")
}

func TestRegisterKind(t *testing.T) {
	c := makeCluster(t, getRandomLocalhostAddr(), "A", "eu-west")
	c.RegisterKind("player", NewPlayer, NewKindConfig())
	c.RegisterKind("inventory", NewInventory, NewKindConfig())
	assert.True(t, c.HasKindLocal("player"))
	assert.True(t, c.HasKindLocal("inventory"))
}

func TestClusterSpawn(t *testing.T) {
	var (
		c1Addr      = getRandomLocalhostAddr()
		c1          = makeCluster(t, c1Addr, "A", "eu-west")
		c2          = makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west")
		wg          = sync.WaitGroup{}
		expectedPID = actor.NewPID(c1Addr, "player/1")
	)

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

	c1.Stop().Wait()
	c2.Stop().Wait()
}

func TestMemberJoin(t *testing.T) {
	c1 := makeCluster(t, getRandomLocalhostAddr(), "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west")
	c2.RegisterKind("player", NewPlayer, NewKindConfig())

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		// we do this so we are 100% sure nodes are connected with eachother.
		case MemberJoinEvent:
			fmt.Println(msg)
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

	c1.Stop().Wait()
	c2.Stop().Wait()
}

func TestActivate(t *testing.T) {
	var (
		addr = getRandomLocalhostAddr()
		c1   = makeCluster(t, addr, "A", "eu-west")
		c2   = makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west")
	)
	c2.RegisterKind("player", NewPlayer, NewKindConfig())

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
				pid := c1.Activate("player", NewActivationConfig().WithID("1"))
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

	c1.Stop().Wait()
	c2.Stop().Wait()
}

func TestDeactivate(t *testing.T) {
	addr := getRandomLocalhostAddr()
	c1 := makeCluster(t, addr, "A", "eu-west")
	c2 := makeCluster(t, getRandomLocalhostAddr(), "B", "eu-west")
	c2.RegisterKind("player", NewPlayer, NewKindConfig())

	expectedPID := actor.NewPID(c2.engine.Address(), "player/1")
	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				pid := c1.Activate("player", NewActivationConfig().WithID("1"))
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

	c1.Stop().Wait()
	c2.Stop().Wait()
}

func TestMemberLeave(t *testing.T) {
	c1Addr := getRandomLocalhostAddr()
	c2Addr := getRandomLocalhostAddr()

	remote := remote.New(c2Addr, nil)
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(remote))
	if err != nil {
		log.Fatal(err)
	}
	config := NewConfig().
		WithID("B").
		WithRegion("eu-east").
		WithEngine(e)
	c2, err := New(config)
	assert.Nil(t, err)

	c1 := makeCluster(t, c1Addr, "A", "eu-west")
	c2.RegisterKind("player", NewPlayer, NewKindConfig())
	c1.Start()

	wg := sync.WaitGroup{}
	wg.Add(1)
	eventPID := c1.engine.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case MemberJoinEvent:
			if msg.Member.ID == "B" {
				remote.Stop().Wait()
			}
		case MemberLeaveEvent:
			assert.Equal(t, msg.Member.ID, c2.ID())
			wg.Done()
		}
	}, "event")
	c1.engine.Subscribe(eventPID)
	c2.Start()

	wg.Wait()
	assert.Equal(t, len(c1.Members()), 1)
	assert.False(t, c1.HasKind("player"))

	c1.Stop().Wait()
	c2.Stop().Wait()
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

func makeCluster(t *testing.T, addr, id, region string) *Cluster {
	config := NewConfig().
		WithID(id).
		WithListenAddr(addr).
		WithRegion(region)
	c, err := New(config)
	assert.Nil(t, err)
	return c
}

func getRandomLocalhostAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", rand.Intn(50000)+10000)
}
