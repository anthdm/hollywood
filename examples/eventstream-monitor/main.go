package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type monitor struct {
	crashes     int
	starts      int
	stops       int
	deadletters int
	msg         string
}

type query struct {
	crashes     int
	starts      int
	stops       int
	deadletters int
}

func newMonitor() actor.Receiver {
	return &monitor{}

}
func (m *monitor) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		c.Engine().Subscribe(c.PID())
	case actor.ActorRestartedEvent:
		m.crashes++
	case actor.ActorStartedEvent:
		m.starts++
	case actor.ActorStoppedEvent:
		m.stops++
	case actor.DeadLetterEvent:
		m.deadletters++
	case query:
		c.Respond(query{
			crashes:     m.crashes,
			starts:      m.starts,
			stops:       m.stops,
			deadletters: m.deadletters,
		})
	}

}

type customMessage struct{}
type unstableActor struct {
	restarts  int
	spawnTime time.Time
}

func newUnstableActor() actor.Receiver {
	return &unstableActor{}
}
func (m *unstableActor) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		m.spawnTime = time.Now()
	case actor.Started:
		fmt.Println("actor started")
	case customMessage:
		if time.Since(m.spawnTime) > time.Second { // We should crash once per second.
			panic("my time has come")
		}

	}
}

func main() {
	e, _ := actor.NewEngine(nil)
	// Spawn a monitor actor and then an unstable actor.
	monitor := e.Spawn(newMonitor, "monitor")
	ua := e.Spawn(newUnstableActor, "unstable_actor", actor.WithMaxRestarts(10000))
	repeater := e.SendRepeat(ua, customMessage{}, time.Millisecond*20)
	time.Sleep(time.Second * 5)
	repeater.Stop()
	e.Poison(ua).Wait()
	res, err := e.Request(monitor, query{}, time.Second).Result()
	if err != nil {
		fmt.Println("Query", err)
	}
	q := res.(query)
	fmt.Printf("Observed %d crashes\n", q.crashes)
	fmt.Printf("Observed %d starts\n", q.starts)
	fmt.Printf("Observed %d stops\n", q.stops)
	fmt.Printf("Observed %d deadletters\n", q.deadletters)
	e.Poison(monitor).Wait() // the monitor will output stats on stop.
	fmt.Println("done")
}
