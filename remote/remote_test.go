package remote

import (
	sync "sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSender(t *testing.T) {
	var (
		a         = makeRemoteEngine("127.0.0.1:4000")
		wg        = sync.WaitGroup{}
		senderPID = actor.NewPID("a", "b")
	)
	pid := a.Spawn(actor.NewTestProducer(t, func(t *testing.T, ctx *actor.Context) {
		if _, ok := ctx.Message().(*TestMessage); ok {
			wg.Done()
			assert.Equal(t, senderPID.Address, ctx.Sender().Address)
			assert.Equal(t, senderPID.ID, ctx.Sender().ID)
		}
	}), "test")

	b := makeRemoteEngine("127.0.0.1:5000")
	wg.Add(1)
	b.SendWithSender(pid, &TestMessage{}, senderPID)
	wg.Wait()
}

func TestRequestResponse(t *testing.T) {
	a := makeRemoteEngine("127.0.0.1:4001")
	pid := a.Spawn(actor.NewTestProducer(t, func(t *testing.T, ctx *actor.Context) {
		if _, ok := ctx.Message().(*TestMessage); ok {
			ctx.Respond(&TestMessage{Data: []byte("foo")})
		}
	}), "test")

	b := makeRemoteEngine("127.0.0.1:5001")
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
