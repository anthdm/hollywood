package main

import (
	"fmt"
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type (
	nameRequest  struct{}
	nameResponse struct {
		name string
	}
)

type nameResponder struct {
	name string
}

func newNameResponder() actor.Receiver {

	return &nameResponder{name: "noname"}
}

func newCustomNameResponder(name string) actor.Producer {
	return func() actor.Receiver {
		return &nameResponder{name}
	}
}

func (r *nameResponder) Receive(ctx *actor.Context) {
	switch ctx.Message().(type) {
	case *nameRequest:
		ctx.Respond(&nameResponse{r.name})
	}
}

// main is the entry point of the application.
// it creates an actor engine and spawns two new actors.
// the first actor is spawned with a default name and the second
// actor is spawned with a custom name, showing you how to pass custom
// arguments to your actors.
func main() {
	e, err := actor.NewEngine(nil)
	if err != nil {
		log.Fatal(err)
	}
	pid := e.Spawn(newNameResponder, "responder")
	// Request a name and block till we got a response or the request
	// timed out.
	resp := e.Request(pid, &nameRequest{}, time.Millisecond)
	// calling Result() will block till we received a response or
	// the request timed out.
	res, err := resp.Result()
	if err != nil {
		log.Fatal(err)
	}
	if name, ok := res.(*nameResponse); ok {
		fmt.Println("received name:", name.name)
	}

	// Spawn a new responder with a custom name.
	pid = e.Spawn(newCustomNameResponder("anthdm"), "custom_responder")
	resp = e.Request(pid, &nameRequest{}, time.Millisecond)
	res, err = resp.Result()
	if err != nil {
		log.Fatal(err)
	}
	if name, ok := res.(*nameResponse); ok {
		fmt.Println("received name:", name.name)
	}
}
