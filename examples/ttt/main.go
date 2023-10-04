package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/stevohuncho/hollywood/actor"
)

func main() {
	e := actor.NewEngine()
	pid := e.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case actor.Started:
			fmt.Println("started")
		case actor.Stopped:
			fmt.Println("stopped")
		default:
			_ = msg
		}
	}, "foobarbas")

	wg := sync.WaitGroup{}
	e.Poison(pid, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}
