package remote

import (
	"fmt"
	sync "sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	var (
		a  = makeRemoteEngine("127.0.0.2:4000")
		b  = makeRemoteEngine("127.0.0.2:5000")
		wg = sync.WaitGroup{}
	)

	// Give remotes time to serve first
	time.Sleep(time.Second)

	wg.Add(1)
	pid := a.Spawn(actor.NewTestProducer(t, func(t *testing.T, ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			wg.Done()
		}
	}), "foo")

	b.Send(pid, &TestMessage{Data: []byte("foo")})
	wg.Wait()
}

func TestWithSender(t *testing.T) {
	var (
		a         = makeRemoteEngine("127.0.0.4:4000")
		b         = makeRemoteEngine("127.0.0.4:5000")
		wg        = sync.WaitGroup{}
		senderPID = actor.NewPID("a", "b")
	)

	// Remote time to serve first
	time.Sleep(time.Second)

	wg.Add(1)
	pid := a.Spawn(actor.NewTestProducer(t, func(t *testing.T, ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case actor.Started:
			fmt.Println("started")
		case actor.Initialized:
			fmt.Println("init")
		case *TestMessage:
			fmt.Println("message")
			assert.Equal(t, senderPID.Address, ctx.Sender().Address)
			assert.Equal(t, senderPID.ID, ctx.Sender().ID)
			wg.Done()
		default:
			fmt.Println(msg)
		}
	}), "test")

	b.SendWithSender(pid, &TestMessage{Data: []byte("f")}, senderPID)
	wg.Wait()
}

func TestRequestResponse(t *testing.T) {
	var (
		a = makeRemoteEngine("127.0.0.1:4001")
		b = makeRemoteEngine("127.0.0.1:5001")
	)
	time.Sleep(time.Second)

	pid := a.Spawn(actor.NewTestProducer(t, func(t *testing.T, ctx *actor.Context) {
		if _, ok := ctx.Message().(*TestMessage); ok {
			ctx.Respond(&TestMessage{Data: []byte("foo")})
		}
	}), "test")

	resp, err := b.Request(pid, &TestMessage{}, time.Second).Result()
	require.Nil(t, err)
	assert.Equal(t, resp.(*TestMessage).Data, []byte("foo"))
}

func makeRemoteEngine(listenAddr string) *actor.Engine {
	e := actor.NewEngine()
	r := New(e, Config{listenAddr})
	e.WithRemote(r)
	return e
}
