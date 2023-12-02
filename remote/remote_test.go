package remote

import (
	"context"
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

func TestSend(t *testing.T) {
	const msgs = 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	aAddr := getRandomLocalhostAddr()
	a, err := makeRemoteEngine(ctx, aAddr)
	assert.NoError(t, err)
	bAddr := getRandomLocalhostAddr()
	b, err := makeRemoteEngine(ctx, bAddr)
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
	wg.Wait()
	cancel()
	// potential race here. when we cancel() the socket is teared down.
	// Is there a nice way to check that the remote is actually shut down?
	err = tcpPing(aAddr)
	fmt.Println("error", err)
	assert.Error(t, err)
	err = tcpPing(bAddr)
	fmt.Println("error", err)
	assert.Error(t, err)

}

func TestWithSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, err := makeRemoteEngine(ctx, getRandomLocalhostAddr())
	assert.NoError(t, err)
	b, err := makeRemoteEngine(ctx, getRandomLocalhostAddr())
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, err := makeRemoteEngine(ctx, getRandomLocalhostAddr())
	assert.NoError(t, err)
	b, err := makeRemoteEngine(ctx, getRandomLocalhostAddr())
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

func makeRemoteEngine(ctx context.Context, listenAddr string) (*actor.Engine, error) {
	e := actor.NewEngine()
	r := New(e, Config{ListenAddr: listenAddr})
	err := e.WithRemote(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("WithRemote: %w", err)
	}
	return e, nil
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
