package remote

import (
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
	var (
		a  = makeRemoteEngine("127.0.0.2:4000")
		b  = makeRemoteEngine("127.0.0.2:5000")
		wg = sync.WaitGroup{}
	)

	wg.Add(10) // send 2 messages
	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			wg.Done()
		}
	}, "dfoo")

	for i := 0; i < 10; i++ {
		b.Send(pid, &TestMessage{Data: []byte("foo")})
	}
	wg.Wait()
}

func TestWithSender(t *testing.T) {
	var (
		a         = makeRemoteEngine("127.0.0.4:4000")
		b         = makeRemoteEngine("127.0.0.4:5000")
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
	// NOTE: Its important when running to remote on the same binary
	// which in the real world will never happen, we need to unlock the OS thread
	// or we will have context deadlines exceeds on request responses.
	// Hence, for testing this we need to set it to 0
	actor.LOCK_OS_THREAD = false
	defer func() {
		actor.LOCK_OS_THREAD = true
	}()
	var (
		a  = makeRemoteEngine("127.0.0.1:4001")
		b  = makeRemoteEngine("127.0.0.1:5001")
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
	r := New(e, Config{listenAddr})
	e.WithRemote(r)
	return e
}
