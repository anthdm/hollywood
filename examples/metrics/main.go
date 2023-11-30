package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PromMetrics struct {
	msgCounter prometheus.Counter
	msgLatency prometheus.Histogram
}

func newPromMetrics(prefix string) *PromMetrics {
	msgCounter := promauto.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_actor_msg_counter", prefix),
		Help: "counter of the messages the actor received",
	})
	msgLatency := promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    fmt.Sprintf("%s_actor_msg_latency", prefix),
		Help:    "actor msg latency",
		Buckets: []float64{0.1, 0.5, 1},
	})
	return &PromMetrics{
		msgCounter: msgCounter,
		msgLatency: msgLatency,
	}
}

func (p *PromMetrics) WithMetrics() func(actor.ReceiveFunc) actor.ReceiveFunc {
	return func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(c *actor.Context) {
			start := time.Now()
			p.msgCounter.Inc()
			next(c)
			ms := time.Since(start).Seconds()
			p.msgLatency.Observe(ms)
		}
	}
}

type Message struct {
	data string
}

type Foo struct{}

func (f *Foo) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		fmt.Println("foo started")
	case actor.Stopped:
	case Message:
		fmt.Println("received:", msg.data)
	}
}

func newFoo() actor.Receiver {
	return &Foo{}
}

type Bar struct{}

func (f *Bar) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case actor.Stopped:
	case Message:
		fmt.Println("received:", msg.data)
	}
}

func newBar() actor.Receiver {
	return &Bar{}
}

func main() {
	promListenAddr := flag.String("promlistenaddr", ":2222", "the listen address of the prometheus http handler")
	flag.Parse()
	go func() {
		http.ListenAndServe(*promListenAddr, promhttp.Handler())
	}()

	var (
		e          = actor.NewEngine()
		foometrics = newPromMetrics("foo")
		barmetrics = newPromMetrics("bar")
		fooPID     = e.Spawn(newFoo, "foo", actor.WithMiddleware(foometrics.WithMetrics()))
		barPID     = e.Spawn(newBar, "bar", actor.WithMiddleware(barmetrics.WithMetrics()))
	)

	for i := 0; i < 10; i++ {
		e.Send(fooPID, Message{data: "message to foo"})
		e.Send(barPID, Message{data: "message to bar"})
		time.Sleep(time.Second)
	}
}
