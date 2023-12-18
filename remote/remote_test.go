package remote

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Needed for now when having the VTProtoserializer
	RegisterType(&TestMessage{})
}

type dlactor struct {
	count int
	n     int
	wg    *sync.WaitGroup
}

func NewDlActor(wg *sync.WaitGroup, n int) actor.Producer {
	return func() actor.Receiver {
		return &dlactor{
			wg: wg,
			n:  n,
		}
	}
}

func (a *dlactor) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.RemoteUnreachableEvent:
		a.wg.Done()
	case actor.DeadLetterEvent:
		a.count++
		if a.count == a.n {
			a.wg.Done()
		}
	}
}

// When a remote is unreachable we retry connecting N times. If the remote is still not reachable
// after N retries, we send all the buffered messages as Deadletter events.
// This test checks if we receive N deadletters after sending N messages to an unreachable
// remote.
func TestRemoteUnreachableMessagesEndUpInDeadletter(t *testing.T) {
	n := 10
	a, _, err := makeRemoteEngine(getRandomLocalhostAddr())
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	pid := a.Spawn(NewDlActor(wg, n), "event")
	a.Subscribe(pid)

	for i := 0; i < n; i++ {
		a.Send(actor.NewPID("127.0.0.1:4000", "foo/bar"), &TestMessage{Data: []byte("foo")})
	}
	wg.Wait()
}

func TestSend(t *testing.T) {
	const msgs = 10
	aAddr := getRandomLocalhostAddr()
	a, ra, err := makeRemoteEngine(aAddr)
	assert.NoError(t, err)
	bAddr := getRandomLocalhostAddr()
	b, rb, err := makeRemoteEngine(bAddr)
	assert.NoError(t, err)
	wg := &sync.WaitGroup{}

	wg.Add(msgs) // send msgs messages
	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			wg.Done()
		}
	}, "dfoo")

	for i := 0; i < msgs; i++ {
		b.Send(pid, &TestMessage{Data: []byte("foo")})
	}
	wg.Wait()        // wait for messages to be received by the actor.
	ra.Stop().Wait() // shutdown the remotes
	rb.Stop().Wait()
	err = tcpPing(aAddr)
	assert.Error(t, err)
	err = tcpPing(bAddr)
	assert.Error(t, err)
}

func TestWithSender(t *testing.T) {
	a, ra, err := makeRemoteEngine(getRandomLocalhostAddr())
	defer ra.Stop()
	assert.NoError(t, err)
	b, rb, err := makeRemoteEngine(getRandomLocalhostAddr())
	defer rb.Stop()
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	senderPID := actor.NewPID("a", "b")

	wg.Add(1)
	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case actor.Started:
		case actor.Initialized:
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			assert.Equal(t, senderPID.Address, c.Sender().Address)
			assert.Equal(t, senderPID.ID, c.Sender().ID)
			wg.Done()
		default:
		}
	}, "test")

	b.SendWithSender(pid, &TestMessage{Data: []byte("foo")}, senderPID)
	wg.Wait()
}

func TestRequestResponse(t *testing.T) {
	a, ra, err := makeRemoteEngine(getRandomLocalhostAddr())
	defer ra.Stop()
	assert.NoError(t, err)
	b, rb, err := makeRemoteEngine(getRandomLocalhostAddr())
	defer rb.Stop()
	assert.NoError(t, err)
	wg := sync.WaitGroup{}

	wg.Add(1)
	pid := a.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		case actor.Started:
			wg.Done()
		case *TestMessage:
			c.Respond(&TestMessage{Data: []byte("foo")})
		}
	}, "test")
	wg.Wait()
	resp, err := b.Request(pid, &TestMessage{Data: []byte("foo")}, time.Second).Result()
	require.Nil(t, err)
	assert.Equal(t, resp.(*TestMessage).Data, []byte("foo"))

	resp, err = b.Request(pid, &TestMessage{Data: []byte("bar")}, time.Second).Result()
	require.Nil(t, err)
	assert.Equal(t, resp.(*TestMessage).Data, []byte("foo"))
}

func TestEventStream(t *testing.T) {
	// Events should work over the wire from the get go.
	// Which is just insane, huh?
	engine, _, err := makeRemoteEngine(getRandomLocalhostAddr())
	assert.NoError(t, err)
	wg := &sync.WaitGroup{}

	wg.Add(2)

	engine.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		case actor.Started:
			c.Engine().Subscribe(c.PID())
		case *TestMessage:
			fmt.Println("actor (a) received event")
			wg.Done()
		}
	}, "actor_a")

	engine.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		case actor.Started:
			c.Engine().Subscribe(c.PID())
		case *TestMessage:
			fmt.Println("actor (b) received event")
			wg.Done()
		}
	}, "actor_b")
	time.Sleep(time.Millisecond)
	engine.BroadcastEvent(&TestMessage{Data: []byte("testevent")})
	wg.Wait()
}

// TestWeird does unexpected things to the remote to see if it panics or freezes.
func TestWeird(t *testing.T) {
	a, ra, err := makeRemoteEngine(getRandomLocalhostAddr())
	if err != nil {
		t.Fatalf("makeRemoteEngine: %v", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	pid := a.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		case actor.Stopped:
			wg.Done()
		}
	}, "weirdactor")
	// let's start the remote once more. this should do nothing.
	err = ra.Start(a)
	assert.Error(t, err)
	err = ra.Start(a)
	assert.Error(t, err)
	err = ra.Start(a)
	assert.Error(t, err)
	// Now stop it a few times to make sure it doesn't freeze or panic:
	ra.Stop().Wait()
	ra.Stop().Wait()
	ra.Stop().Wait()
	a.Poison(pid) // poison the actor. this doesn't go via the remote, so it should be fine.
	wg.Wait()     // wait for the actor to stop.
}

func makeRemoteEngine(listenAddr string) (*actor.Engine, *Remote, error) {
	var e *actor.Engine
	r := New(Config{ListenAddr: listenAddr})
	var err error
	e, err = actor.NewEngine(actor.EngineOptRemote(r))
	if err != nil {
		return nil, nil, fmt.Errorf("actor.NewEngine: %w", err)
	}
	return e, r, nil
}

func getRandomLocalhostAddr() string {
	return fmt.Sprintf("localhost:%d", rand.Intn(50000)+10000)
}

func tcpPing(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	return nil
}
