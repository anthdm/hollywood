package main

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

type Message struct{}

/*
func MeasureFunctionPerformance(function interface{}, args ...interface{}) (duration time.Duration, memoryAllocated uint64, results []interface{}) {
	// Convert the function to a reflect.Value
	funcValue := reflect.ValueOf(function)
	if funcValue.Kind() != reflect.Func {
		panic("MeasureFunctionPerformance requires a function as the first argument")
	}
	// Convert the arguments to reflect.Values
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// Force a garbage collection for a clean memory state
	runtime.GC()

	// Get memory stats before execution
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	startMem := m1.Alloc

	// Note the start time
	startTime := time.Now()

	// Call the function using reflection
	out := funcValue.Call(in)

	// Note the end time
	endTime := time.Now()

	// Get memory stats after execution
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	endMem := m2.Alloc

	// Calculate time and memory consumed
	duration = endTime.Sub(startTime)
	memoryAllocated = endMem - startMem

	// Convert the return values to a slice of interfaces
	results = make([]interface{}, len(out))
	for i, r := range out {
		results[i] = r.Interface()
	}
	return
}
*/

func makeEngines(noOfEngines int) []*actor.Engine {
	engines := make([]*actor.Engine, noOfEngines)

	for i := 0; i < noOfEngines; i++ {
		r := remote.New(remote.Config{ListenAddr: fmt.Sprintf("localhost:%d", 3000+i)})
		e, err := actor.NewEngine(actor.EngineOptRemote(r))
		if err != nil {
			log.Fatal(err)
		}
		engines[i] = e
	}
	return engines
}

func makeActors(engines []*actor.Engine, actorsPerEngine int, counter *atomic.Int64) [][]*actor.PID {
	actors := make([][]*actor.PID, actorsPerEngine)
	for i := 0; i < len(engines); i++ {
		actors[i] = make([]*actor.PID, actorsPerEngine)
		for j := 0; j < actorsPerEngine; j++ {
			actors[i][j] = engines[i].SpawnFunc(func(c *actor.Context) {
				switch c.Message().(type) {
				case string:
					counter.Add(1)
				}
			}, strconv.Itoa(j))
		}
	}
	return actors
}

// sendMessages will take a list of actors and send messages to them
// it'll spin up 10 goroutines that will send messages to the actors.
// each goroutine will send messages to a random actor from another engine
// this will simulate a real world scenario where actors are distributed
// across multiple engines. Once the duration has passed, the function will
// return
func sendMessages(actorList [][]*actor.PID, engineList []*actor.Engine, d time.Duration, maxActor int) int64 {
	msgCount := &atomic.Int64{}
	wg := &sync.WaitGroup{}

	deadline := time.Now().Add(d)
	for i, list := range actorList {
		wg.Add(1)
		go func(i int, list []*actor.PID) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				// Select a random engine, different from the current one
				engineIdx := notQuiteRandom(int32(len(engineList)), int32(i))
				engine := engineList[engineIdx]
				// Select a random actor from the list
				targetIdx := rand.Intn(maxActor)
				target := list[targetIdx]
				// Send a message to the actor
				engine.Send(target, &Message{})
			}
		}(i, list)
	}

	wg.Wait()
	fmt.Printf("Total messages sent: %d\n", msgCount.Load())
	return msgCount.Load()
}

func main() {
	const (
		actorsPerEngine   = 10
		engines           = 10
		messagesPerEngine = 10_000
	)
	counter := &atomic.Int64{}

	if runtime.GOMAXPROCS(runtime.NumCPU()) == 1 {
		slog.Error("GOMAXPROCS must be greater than 1")
		os.Exit(1)
	}
	lh := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(lh)
	fmt.Println("spawning engines")
	engineList := makeEngines(engines)
	actorList := makeActors(engineList, actorsPerEngine, counter)
	sendMessages(actorList, engineList, time.Second*10, actorsPerEngine)

	// now send messages. We will send 10_000 messages to each actor
}

// notQuiteRandom will generate a random number between 0 and n
// but it will avoid the number avoid
func notQuiteRandom(n, avoid int32) int32 {
	for {
		r := rand.Int31n(n)
		if r != avoid {
			return r
		}
	}
}
