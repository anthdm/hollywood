package main

import (
	"errors"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
	"log/slog"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative message.proto

type monitor struct {
}

func (m *monitor) Receive(ctx *actor.Context) {
	switch ctx.Message().(type) {
	case actor.Initialized:
		ctx.Engine().Subscribe(ctx.PID())
	case actor.DeadLetterEvent:
		deadLetters.Add(1)
	}
}
func newMonitor() actor.Receiver {
	return &monitor{}
}

type benchMarkActor struct {
	internalMessageCount int64
}

var (
	receiveCount *atomic.Int64
	sendCount    *atomic.Int64
	deadLetters  *atomic.Int64
)

func init() {
	receiveCount = &atomic.Int64{}
	sendCount = &atomic.Int64{}
	deadLetters = &atomic.Int64{}
}

func (b *benchMarkActor) Receive(ctx *actor.Context) {
	switch ctx.Message().(type) {
	case *Message:
		b.internalMessageCount++
		receiveCount.Add(1)
	case *Ping:
		ctx.Respond(&Pong{})
	}
}

func newActor() actor.Receiver {
	return &benchMarkActor{}
}

type Benchmark struct {
	engineCount     int
	actorsPerEngine int
	senders         int
	engines         []*Engine
}

func (b *Benchmark) randomEngine() *Engine {
	return b.engines[rand.Intn(len(b.engines))]
}

type Engine struct {
	engineID      int
	actors        []*actor.PID
	engine        *actor.Engine
	targetEngines []*Engine
	monitor       *actor.PID
}

func (e *Engine) randomActor() *actor.PID {
	return e.actors[rand.Intn(len(e.actors))]
}
func (e *Engine) randomTargetEngine() *Engine {
	return e.targetEngines[rand.Intn(len(e.targetEngines))]
}

func newBenchmark(engineCount, actorsPerEngine, senders int) *Benchmark {
	b := &Benchmark{
		engineCount:     engineCount,
		actorsPerEngine: actorsPerEngine,
		engines:         make([]*Engine, engineCount),
		senders:         senders,
	}
	return b
}
func (b *Benchmark) spawnEngines() error {
	for i := 0; i < b.engineCount; i++ {
		r := remote.New(remote.Config{ListenAddr: fmt.Sprintf("localhost:%d", 4000+i)})
		e, err := actor.NewEngine(actor.EngineOptRemote(r))
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}
		// spawn the monitor
		b.engines[i] = &Engine{
			engineID: i,
			actors:   make([]*actor.PID, b.actorsPerEngine),
			engine:   e,
			monitor:  e.Spawn(newMonitor, "monitor"),
		}

	}
	// now set up the target engines. These are pointers to all the other engines, except the current one.
	for i := 0; i < b.engineCount; i++ {
		for j := 0; j < b.engineCount; j++ {
			if i == j {
				continue
			}
			b.engines[i].targetEngines = append(b.engines[i].targetEngines, b.engines[j])
		}
	}
	fmt.Printf("spawned %d engines\n", b.engineCount)
	return nil
}

func (b *Benchmark) spawnActors() error {
	for i := 0; i < b.engineCount; i++ {
		for j := 0; j < b.actorsPerEngine; j++ {
			id := fmt.Sprintf("engine-%d-actor-%d", i, j)
			b.engines[i].actors[j] = b.engines[i].engine.Spawn(newActor, id)
		}
	}
	fmt.Printf("spawned %d actors per engine\n", b.actorsPerEngine)
	return nil
}
func (b *Benchmark) sendMessages(d time.Duration) error {
	wg := sync.WaitGroup{}
	wg.Add(b.senders)
	deadline := time.Now().Add(d)
	for i := 0; i < b.senders; i++ {
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				// pick a random engine to send from
				engine := b.randomEngine()
				// pick a random target engine:
				targetEngine := engine.randomTargetEngine()
				// pick a random target actor from the engine
				targetActor := targetEngine.randomActor()
				// send the message
				engine.engine.Send(targetActor, &Message{})
				sendCount.Add(1)
			}
		}()
	}
	wg.Wait()
	time.Sleep(time.Millisecond * 1000) // wait for the messages to be delivered
	// compare the global send count with the receive count
	if sendCount.Load() != receiveCount.Load() {
		return fmt.Errorf("send count and receive count does not match: %d != %d", sendCount.Load(), receiveCount.Load())
	}
	return nil
}

func benchmark() error {
	const (
		engines         = 10
		actorsPerEngine = 2000
		senders         = 20
		duration        = time.Second * 10
	)

	if runtime.GOMAXPROCS(runtime.NumCPU()) == 1 {
		return errors.New("GOMAXPROCS must be greater than 1")
	}
	lh := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(lh)

	benchmark := newBenchmark(engines, actorsPerEngine, senders)
	err := benchmark.spawnEngines()
	if err != nil {
		return fmt.Errorf("failed to spawn engines: %w", err)
	}
	err = benchmark.spawnActors()
	if err != nil {
		return fmt.Errorf("failed to spawn actors: %w", err)
	}
	repCh := make(chan struct{})
	go func() {
		lastSendCount := sendCount.Load()
		for {
			select {
			case <-repCh:
				return
			case <-time.After(time.Second):
				fmt.Printf("Messages sent per second %d\n", sendCount.Load()-lastSendCount)
				lastSendCount = sendCount.Load()
			}
		}
	}()
	fmt.Printf("Send storm starting, will send for %v using %d workers\n", duration, senders)
	err = benchmark.sendMessages(duration)
	if err != nil {
		return fmt.Errorf("failed to send messages: %w", err)
	}
	close(repCh)
	fmt.Printf("Concurrent senders: %d messages sent %d, messages received %d - duration: %v\n", senders, sendCount.Load(), receiveCount.Load(), duration)
	fmt.Printf("messages per second: %d\n", receiveCount.Load()/int64(duration.Seconds()))
	fmt.Printf("deadletters: %d\n", deadLetters.Load())
	return nil
}

func main() {
	err := benchmark()
	if err != nil {
		slog.Error("failed to run benchmark", "err", err)
		os.Exit(1)
	}

}
