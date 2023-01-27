package main

import (
	"fmt"
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type (
	nameRequest struct{}

	nameResponse struct {
		name string
	}
)

type nameResponder struct{}

func newNameResponder() actor.Receiver {
	return &nameResponder{}
}

func (r *nameResponder) Receive(ctx *actor.Context) {
	switch ctx.Message().(type) {
	case *nameRequest:
		ctx.Respond(&nameResponse{name: "anthdm"})
	}
}

func main() {
	e := actor.NewEngine()
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
	fmt.Println(res)
}
