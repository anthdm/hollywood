package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

func makeRemoteEngine(addr string) *actor.Engine {
	e := actor.NewEngine()
	r := remote.New(e, remote.Config{ListenAddr: addr})

	err := e.WithRemote(context.Background(), r)
	if err != nil {
		slog.Error("WithRemote", "error", err)
		os.Exit(1)
	}
	return e
}

func benchmarkRemote() {
	var (
		a    = makeRemoteEngine("127.0.0.1:3000")
		b    = makeRemoteEngine("127.0.0.1:3001")
		pidB = b.SpawnFunc(func(c *actor.Context) {}, "bench", actor.WithInboxSize(1024*8))
	)
	its := []int{
		1_000_000,
		10_000_000,
	}
	for i := 0; i < len(its); i++ {
		start := time.Now()
		for j := 0; j < its[i]; j++ {
			a.Send(pidB, pidB)
		}
		fmt.Printf("[BENCH HOLLYWOOD REMOTE] processed %d messages in %v\n", its[i], time.Since(start))
	}
}

func benchmarkLocal() {
	e := actor.NewEngine()
	pid := e.SpawnFunc(func(c *actor.Context) {}, "bench", actor.WithInboxSize(1024*8))
	its := []int{
		1_000_000,
		10_000_000,
	}
	payload := make([]byte, 128)
	for i := 0; i < len(its); i++ {
		start := time.Now()
		for j := 0; j < its[i]; j++ {
			e.Send(pid, payload)
		}
		fmt.Printf("[BENCH HOLLYWOOD LOCAL] processed %d messages in %v\n", its[i], time.Since(start))
	}
}

func main() {
	if runtime.GOMAXPROCS(runtime.NumCPU()) == 1 {
		slog.Error("GOMAXPROCS must be greater than 1")
		os.Exit(1)
	}
	benchmarkLocal()
	benchmarkRemote()
}
