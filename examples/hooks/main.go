package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type hookReceiver struct{}

func newHookReceiver() actor.Receiver {
	return &hookReceiver{}
}

func (h *hookReceiver) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started, actor.Stopped:
	default:
		fmt.Println("received: ", reflect.TypeOf(msg))
	}
}

func (h *hookReceiver) OnStart(ctx *actor.Context) {
	fmt.Println("started from hooks, my PID: ", ctx.PID())
}

func (h *hookReceiver) OnStop(ctx *actor.Context) {
	fmt.Println("the actor has stopped from hooks")
}

func main() {
	actor.PIDSeparator = "→"
	e := actor.NewEngine()
	pid := e.SpawnOpts(actor.Opts{
		Producer: newHookReceiver,
		Name:     "foo",
		// WithHooks set to true will give your receiver
		// the ability to use the OnStarted and OnStopped hooks.
		// NOTE: these will need to be implemented or the engine will panic
		WithHooks: true,
	})
	time.Sleep(time.Millisecond)
	e.Poison(pid)
	time.Sleep(time.Second)
}
