package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

const (
	min = 1
	max = 300
)

type handleWithPid func(ws *websocket.Conn) (*actor.PID, *chan struct{})

func HandleFunc(f handleWithPid) websocket.Handler {
	return func(c *websocket.Conn) {
		defer fmt.Println("web socket is deleted")

		// We get QuitChannel. If we can't write later and we get a write error, we break this scope (websocket).
		pid, quitCh := f(c)
		fmt.Println("new websocket process is spawned: ", pid.ID)

		for {
			// Waiting for break call.
			<-*quitCh

			// Kill the websocket.
			break
		}
	}
}

func GenerateProcessForWs(ws *websocket.Conn) (*actor.PID, *chan struct{}) {
	// Generate unique process id for new process.
	now := time.Now()
	rand.Seed(now.UnixNano())
	randNum := rand.Intn(max-min+1) + min
	salt := strconv.Itoa(randNum + now.Nanosecond())

	// Spawn new pid for new socket.
	pid := engine.Spawn(webSocketFoo, salt)

	// Create a channel to break the socket.
	quitCh := make(chan struct{})

	// Send datas which is init values.
	engine.Send(pid, &setWsVal{
		ws:     ws,
		quitCh: &quitCh,
	})

	return pid, &quitCh
}
