package remote

import (
	"fmt"
	"github.com/anthdm/hollywood/log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	debugLog = true // if you want a lot of noise when debugging the tests set this to true.
)

func init() {
	// Needed for now when having the VTProtoserializer
	RegisterType(&TestMessage{})
}

func TestSend(t *testing.T) {
	const msgs = 10
	aAddr := getRandomLocalhostAddr()
	a, ra, err := makeRemoteEngine(aAddr)
	assert.NoError(t, err)
	bAddr := getRandomLocalhostAddr()
	b, rb, err := makeRemoteEngine(bAddr)
	assert.NoError(t, err)
	wg := sync.WaitGroup{}

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

func makeRemoteEngine(listenAddr string) (*actor.Engine, *Remote, error) {
	var e *actor.Engine
	switch debugLog {
	case false:
		e = actor.NewEngine()
	case true:
		e = actor.NewEngine(actor.EngineOptLogger(log.Debug()))
	}
	r := New(e, Config{ListenAddr: listenAddr})
	err := e.WithRemote(r)
	if err != nil {
		return nil, nil, fmt.Errorf("WithRemote: %w", err)
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
