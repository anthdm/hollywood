package remote

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// NatsConfig holds the configuration for a NATS remote
type NatsConfig struct {
	natsURL   string
	tlsConfig *tls.Config
	creds     nats.Option
	// SubjectPrefix is used for node subjects. Defaults to "hollywood.node"
	// when empty.
	subjectPrefix string
}

// WithTLS configures TLS for the NATS connection
func (c NatsConfig) WithTLS(tlsconf *tls.Config) NatsConfig {
	c.tlsConfig = tlsconf
	return c
}

// WithCredentials sets the NATS connection credentials or NKeys
func (c NatsConfig) WithCredentials(userOrChainedFile string, seedFiles ...string) NatsConfig {
	c.creds = nats.UserCredentials(userOrChainedFile, seedFiles...)
	return c
}

// WithURL sets the URL of the nats broker
func (c NatsConfig) WithURL(addr string) NatsConfig {
	c.natsURL = addr
	return c
}

// WithSubjectPrefix sets the NATS subject prefix for node addressing.
func (c NatsConfig) WithSubjectPrefix(prefix string) NatsConfig {
	c.subjectPrefix = prefix
	return c
}

type NatsRemote struct {
	addr         string
	nc           *nats.Conn
	engine       *actor.Engine
	config       NatsConfig
	writerPID    *actor.PID
	subscription *nats.Subscription
	state        atomic.Uint32
}

type natsWriterShutdown struct{}

// NewNats creates a new "Remote" object that communicates over NATS given a Config.
func NewNats(addr string, config NatsConfig) *NatsRemote {
	r := &NatsRemote{
		addr:   addr,
		config: config,
	}
	r.state.Store(stateInitialized)
	return r
}

// Start starts the Nats Remote
func (r *NatsRemote) Start(e *actor.Engine) error {
	if r.state.Load() != stateInitialized {
		return fmt.Errorf("nats remote already started")
	}
	r.engine = e
	var err error
	natsURL := r.config.natsURL
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	slog.Debug("nats connecting", "url", natsURL, "addr", r.addr)
	switch r.config.tlsConfig {
	case nil:
		switch r.config.creds {
		case nil:
			r.nc, err = nats.Connect(natsURL, nats.Name(r.addr))
		default:
			r.nc, err = nats.Connect(natsURL, nats.Name(r.addr), r.config.creds)
		}
	default:
		switch r.config.creds {
		case nil:
			r.nc, err = nats.Connect(natsURL, nats.Name(r.addr), nats.Secure(r.config.tlsConfig))
		default:
			r.nc, err = nats.Connect(natsURL, nats.Name(r.addr), nats.Secure(r.config.tlsConfig), r.config.creds)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to start NATS client: %w", err)
	}

	sub, err := r.nc.Subscribe(r.subjectForAddress(r.addr), r.handleInbound)
	if err != nil {
		_ = r.nc.Drain()
		r.nc.Close()
		return fmt.Errorf("failed to subscribe to node subject: %w", err)
	}
	if err := r.nc.FlushTimeout(2 * time.Second); err != nil {
		_ = sub.Unsubscribe()
		_ = r.nc.Drain()
		r.nc.Close()
		return fmt.Errorf("failed to flush nats subscription: %w", err)
	}
	if err := r.nc.LastError(); err != nil {
		_ = sub.Unsubscribe()
		_ = r.nc.Drain()
		r.nc.Close()
		return fmt.Errorf("nats connection error after subscribe: %w", err)
	}
	r.subscription = sub
	r.writerPID = r.engine.SpawnProc(newNatsWriter(r))
	r.state.Store(stateRunning)
	return nil
}

// Stop disconnects the Nats Remote
func (r *NatsRemote) Stop() *sync.WaitGroup {
	if r.state.Load() != stateRunning {
		slog.Warn("remote already stopped but stop was called", "state", r.state.Load())
		return &sync.WaitGroup{}
	}
	r.state.Store(stateStopped)
	if r.writerPID != nil {
		r.engine.Send(r.writerPID, natsWriterShutdown{})
	}
	if r.subscription != nil {
		if err := r.subscription.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrBadSubscription) {
			slog.Error("error unsubscribing nats subscription", "err", err)
		}
	}
	if err := r.nc.Drain(); err != nil {
		slog.Error("error draining nats connextion", "err", err)
	}
	r.nc.Close()
	return &sync.WaitGroup{}
}

// Address returns the listen address of the Nats remote.
func (r *NatsRemote) Address() string {
	return r.addr
}

// Send sends the given message to the process with the given pid over the network.
// Optional a "Sender PID" can be given to inform the receiving process who sent the
// message.
func (r *NatsRemote) Send(pid *actor.PID, msg any, sender *actor.PID) {
	r.engine.Send(r.writerPID, &streamDeliver{
		target: pid,
		sender: sender,
		msg:    msg,
	})
}

func (r *NatsRemote) subjectForAddress(address string) string {
	prefix := r.config.subjectPrefix
	if prefix == "" {
		prefix = "hollywood.node"
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(address))
	return fmt.Sprintf("%s.%s", prefix, encoded)
}

func (r *NatsRemote) handleInbound(msg *nats.Msg) {
	if r.state.Load() != stateRunning {
		return
	}
	env := &Envelope{}
	if err := proto.Unmarshal(msg.Data, env); err != nil {
		slog.Error("failed to decode nats envelope", "err", err, "subject", msg.Subject)
		return
	}

	for _, m := range env.Messages {
		if m == nil {
			continue
		}
		if int(m.TypeNameIndex) >= len(env.TypeNames) || int(m.TargetIndex) >= len(env.Targets) {
			slog.Error("received malformed nats envelope indexes", "subject", msg.Subject)
			continue
		}
		tname := env.TypeNames[m.TypeNameIndex]
		payload, err := ProtoSerializer{}.Deserialize(m.Data, tname)
		if err != nil {
			slog.Error("failed to deserialize nats payload", "err", err, "type", tname)
			continue
		}
		target := env.Targets[m.TargetIndex]
		var sender *actor.PID
		if len(env.Senders) > 0 && int(m.SenderIndex) < len(env.Senders) {
			sender = env.Senders[m.SenderIndex]
		}
		r.engine.SendLocal(target, payload, sender)
	}
}

type natsWriter struct {
	remote *NatsRemote
	pid    *actor.PID
	inbox  actor.Inboxer
}

func newNatsWriter(r *NatsRemote) actor.Processer {
	return &natsWriter{
		remote: r,
		pid:    actor.NewPID(r.addr, "nats_writer"),
		inbox:  actor.NewInbox(streamWriterBatchSize),
	}
}

func (w *natsWriter) Start() {
	w.inbox.Start(w)
}

func (w *natsWriter) PID() *actor.PID { return w.pid }

func (w *natsWriter) Send(_ *actor.PID, msg any, sender *actor.PID) {
	w.inbox.Send(actor.Envelope{Msg: msg, Sender: sender})
}

func (w *natsWriter) Invoke(msgs []actor.Envelope) {
	type addrBatch struct {
		typeLookup   map[string]int32
		typeNames    []string
		senderLookup map[uint64]int32
		senders      []*actor.PID
		targetLookup map[uint64]int32
		targets      []*actor.PID
		messages     []*Message
		deliveries   []*streamDeliver
	}

	batches := map[string]*addrBatch{}
	for i := range msgs {
		if _, ok := msgs[i].Msg.(natsWriterShutdown); ok {
			w.Shutdown()
			return
		}
		deliver, ok := msgs[i].Msg.(*streamDeliver)
		if !ok || deliver == nil || deliver.target == nil {
			continue
		}
		addr := deliver.target.Address
		batch, exists := batches[addr]
		if !exists {
			batch = &addrBatch{
				typeLookup:   make(map[string]int32),
				typeNames:    make([]string, 0, 16),
				senderLookup: make(map[uint64]int32),
				senders:      make([]*actor.PID, 0, 16),
				targetLookup: make(map[uint64]int32),
				targets:      make([]*actor.PID, 0, 16),
				messages:     make([]*Message, 0, len(msgs)),
				deliveries:   make([]*streamDeliver, 0, len(msgs)),
			}
			batches[addr] = batch
		}

		typeID, typeNames := lookupTypeName(batch.typeLookup, ProtoSerializer{}.TypeName(deliver.msg), batch.typeNames)
		batch.typeNames = typeNames
		senderID, senders := lookupPIDs(batch.senderLookup, deliver.sender, batch.senders)
		batch.senders = senders
		targetID, targets := lookupPIDs(batch.targetLookup, deliver.target, batch.targets)
		batch.targets = targets

		b, err := ProtoSerializer{}.Serialize(deliver.msg)
		if err != nil {
			slog.Error("nats serialize", "err", err)
			continue
		}
		batch.messages = append(batch.messages, &Message{
			Data:          b,
			TypeNameIndex: typeID,
			SenderIndex:   senderID,
			TargetIndex:   targetID,
		})
		batch.deliveries = append(batch.deliveries, deliver)
	}

	if w.remote.nc.Status() != nats.CONNECTED {
		for addr, batch := range batches {
			if len(batch.deliveries) == 0 {
				continue
			}
			w.onUnreachable(addr, batch.deliveries)
		}
		return
	}

	for addr, batch := range batches {
		if len(batch.messages) == 0 {
			continue
		}
		env := &Envelope{
			Senders:   batch.senders,
			Targets:   batch.targets,
			TypeNames: batch.typeNames,
			Messages:  batch.messages,
		}
		payload, err := proto.Marshal(env)
		if err != nil {
			slog.Error("failed to marshal nats envelope", "err", err)
			continue
		}
		subject := w.remote.subjectForAddress(addr)
		if err := w.remote.nc.Publish(subject, payload); err != nil {
			slog.Error("failed to publish nats envelope", "err", err, "subject", subject)
			w.onUnreachable(addr, batch.deliveries)
		}
	}
}

func (w *natsWriter) onUnreachable(addr string, deliveries []*streamDeliver) {
	w.remote.engine.BroadcastEvent(actor.RemoteUnreachableEvent{ListenAddr: addr})
	for _, d := range deliveries {
		if d == nil {
			continue
		}
		w.remote.engine.BroadcastEvent(actor.DeadLetterEvent{
			Target:  d.target,
			Message: d.msg,
			Sender:  d.sender,
		})
	}
}

func (w *natsWriter) Shutdown() {
	_ = w.inbox.Stop()
	w.remote.engine.Registry.Remove(w.PID())
}
