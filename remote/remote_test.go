package remote

import (
	"fmt"
	"math/rand"
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

func TestSend(t *testing.T) {
	const msgs = 10
	var (
		a  = makeRemoteEngine(getRandomLocalhostAddr())
		b  = makeRemoteEngine(getRandomLocalhostAddr())
		wg = sync.WaitGroup{}
	)

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
	wg.Wait()
}

func TestWithSender(t *testing.T) {
	var (
		a         = makeRemoteEngine(getRandomLocalhostAddr())
		b         = makeRemoteEngine(getRandomLocalhostAddr())
		wg        = sync.WaitGroup{}
		senderPID = actor.NewPID("a", "b")
	)

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
	var (
		a  = makeRemoteEngine(getRandomLocalhostAddr())
		b  = makeRemoteEngine(getRandomLocalhostAddr())
		wg = sync.WaitGroup{}
	)

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

func makeRemoteEngine(listenAddr string) *actor.Engine {
	e := actor.NewEngine()
	r := New(e, Config{ListenAddr: listenAddr})
	e.WithRemote(r)
	return e
}

func getRandomLocalhostAddr() string {
	return fmt.Sprintf("localhost:%d", rand.Intn(50000)+10000)
}
