package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
)

func benchmarkLocal() {
	e := actor.NewEngine()
	pid := e.SpawnFunc(func(c *actor.Context) {}, "bench")
	its := []int{
		1000_000,
		1000_000_0,
		1000_000_00,
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
	benchmarkLocal()
}
