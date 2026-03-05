package remote

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNatsRemoteSend(t *testing.T) {
	ts := startEmbeddedNATSServer(t)
	defer ts.Close(t)

	const msgs = 16
	aAddr := fmt.Sprintf("nats-node-a-%d", time.Now().UnixNano())
	bAddr := fmt.Sprintf("nats-node-b-%d", time.Now().UnixNano())

	a, ra := makeNatsEngine(t, aAddr, ts.ClientURL())
	defer func() { ra.Stop().Wait() }()
	b, rb := makeNatsEngine(t, bAddr, ts.ClientURL())
	defer func() { rb.Stop().Wait() }()

	wg := &sync.WaitGroup{}
	wg.Add(msgs)

	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, []byte("foo"), msg.Data)
			wg.Done()
		}
	}, "receiver")

	for range msgs {
		b.Send(pid, &TestMessage{Data: []byte("foo")})
	}

	waitTimeout(t, wg, 5*time.Second)
}

func TestNatsRemoteWithSender(t *testing.T) {
	ts := startEmbeddedNATSServer(t)
	defer ts.Close(t)

	a, ra := makeNatsEngine(t, "nats-withsender-a", ts.ClientURL())
	defer func() { ra.Stop().Wait() }()
	b, rb := makeNatsEngine(t, "nats-withsender-b", ts.ClientURL())
	defer func() { rb.Stop().Wait() }()

	senderPID := actor.NewPID("custom-address", "custom-id")
	wg := &sync.WaitGroup{}
	wg.Add(1)

	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, []byte("foo"), msg.Data)
			require.NotNil(t, c.Sender())
			assert.Equal(t, senderPID.Address, c.Sender().Address)
			assert.Equal(t, senderPID.ID, c.Sender().ID)
			wg.Done()
		}
	}, "receiver")

	b.SendWithSender(pid, &TestMessage{Data: []byte("foo")}, senderPID)
	waitTimeout(t, wg, 5*time.Second)
}

func TestNatsRemoteRequestResponse(t *testing.T) {
	ts := startEmbeddedNATSServer(t)
	defer ts.Close(t)

	a, ra := makeNatsEngine(t, "nats-req-a", ts.ClientURL())
	defer func() { ra.Stop().Wait() }()
	b, rb := makeNatsEngine(t, "nats-req-b", ts.ClientURL())
	defer func() { rb.Stop().Wait() }()

	started := &sync.WaitGroup{}
	started.Add(1)

	pid := a.SpawnFunc(func(c *actor.Context) {
		switch c.Message().(type) {
		case actor.Started:
			started.Done()
		case *TestMessage:
			c.Respond(&TestMessage{Data: []byte("ok")})
		}
	}, "responder")
	waitTimeout(t, started, 3*time.Second)

	resp, err := b.Request(pid, &TestMessage{Data: []byte("ping")}, 3*time.Second).Result()
	require.NoError(t, err)
	require.IsType(t, &TestMessage{}, resp)
	assert.Equal(t, []byte("ok"), resp.(*TestMessage).Data)
}

func TestNatsRemoteStopIdempotent(t *testing.T) {
	ts := startEmbeddedNATSServer(t)
	defer ts.Close(t)

	_, r := makeNatsEngine(t, "nats-stop", ts.ClientURL())

	r.Stop().Wait()
	r.Stop().Wait()
	r.Stop().Wait()
}

func TestNatsRemoteUnreachableMessagesEndUpInDeadletter(t *testing.T) {
	const n = 10
	ts := startEmbeddedNATSServer(t)

	a, ra := makeNatsEngine(t, "nats-unreachable-a", ts.ClientURL())
	defer func() { ra.Stop().Wait() }()

	wg := &sync.WaitGroup{}
	wg.Add(2) // one unreachable event and one deadletter completion

	pid := a.Spawn(NewDlActor(wg, n), "event")
	a.Subscribe(pid)

	// Simulate broker outage after engine start.
	ts.Close(t)

	// Wait for connection status transition away from CONNECTED.
	waitUntil(t, 3*time.Second, func() bool {
		return ra.nc == nil || ra.nc.Status() != nats.CONNECTED
	})

	target := actor.NewPID("nats-unreachable-b", "foo/bar")
	for i := 0; i < n; i++ {
		a.Send(target, &TestMessage{Data: []byte("foo")})
	}

	waitTimeout(t, wg, 5*time.Second)
}

func TestNatsRemoteInboundManualPublish(t *testing.T) {
	ts := startEmbeddedNATSServer(t)
	defer ts.Close(t)

	a, ra := makeNatsEngine(t, "nats-inbound-a", ts.ClientURL())
	defer func() { ra.Stop().Wait() }()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	pid := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, []byte("manual"), msg.Data)
			wg.Done()
		}
	}, "receiver")

	env := &Envelope{
		TypeNames: []string{ProtoSerializer{}.TypeName(&TestMessage{})},
		Targets:   []*actor.PID{pid},
		Messages: []*Message{
			{
				Data:          mustSerializeTestMessage(t, &TestMessage{Data: []byte("manual")}),
				TargetIndex:   0,
				SenderIndex:   0,
				TypeNameIndex: 0,
			},
		},
	}
	b, err := proto.Marshal(env)
	require.NoError(t, err)

	nc, err := nats.Connect(ts.ClientURL())
	require.NoError(t, err)
	defer nc.Close()
	spySub, err := nc.SubscribeSync(ra.subjectForAddress(ra.addr))
	require.NoError(t, err)
	err = nc.FlushTimeout(2 * time.Second)
	require.NoError(t, err)

	err = nc.Publish(ra.subjectForAddress(ra.addr), b)
	require.NoError(t, err)
	err = nc.FlushTimeout(2 * time.Second)
	require.NoError(t, err)
	_, err = spySub.NextMsg(2 * time.Second)
	require.NoError(t, err)

	waitTimeout(t, wg, 5*time.Second)
}

func makeNatsEngine(t *testing.T, nodeAddr, natsURL string) (*actor.Engine, *NatsRemote) {
	t.Helper()

	r := NewNats(nodeAddr, NatsConfig{}.WithURL(natsURL))
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(r))
	require.NoError(t, err)
	return e, r
}

type testNATSServer struct {
	srv *nserver.Server
}

func startEmbeddedNATSServer(t *testing.T) *testNATSServer {
	t.Helper()

	opts := &nserver.Options{
		Host: "127.0.0.1",
		Port: -1, // ask server to pick a free ephemeral port
	}
	srv, err := nserver.NewServer(opts)
	require.NoError(t, err)

	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		srv.Shutdown()
		t.Fatal("embedded NATS server did not become ready")
	}

	return &testNATSServer{srv: srv}
}

func (s *testNATSServer) ClientURL() string {
	return s.srv.ClientURL()
}

func (s *testNATSServer) Close(t *testing.T) {
	t.Helper()
	if s == nil || s.srv == nil {
		return
	}
	s.srv.Shutdown()
	s.srv.WaitForShutdown()
}

func waitTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("timeout after %s waiting for waitgroup", timeout)
	}
}

func mustSerializeTestMessage(t *testing.T, msg *TestMessage) []byte {
	t.Helper()
	b, err := ProtoSerializer{}.Serialize(msg)
	require.NoError(t, err)
	return b
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
