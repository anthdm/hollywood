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
	MIN = 1
	MAX = 300
)

type handleWithPid func(ws *websocket.Conn) (*actor.PID, *chan struct{})

func HandleFunc(f handleWithPid) websocket.Handler {
	return func(c *websocket.Conn) {
		defer fmt.Println("web socket is deleted")
		// If we can't write to the socket later and we get-
		// a "write error", we break this (websocket) scope.
		//
		// Generating   a new goblin-process    for   websocket.
		// Getting a quitChanel to break websocket handle scope.
		pid, quitCh := f(c)
		fmt.Println("new goblin-process which is holding incoming websocket is spawned: ", pid.ID)

		for {
			// Waiting for  the "break call"  from goblins-processes.
			<-*quitCh

			// Kill the websocket handling scope.
			break
		}
	}
}

func GenerateProcessForWs(ws *websocket.Conn) (*actor.PID, *chan struct{}) {
	// Create unique pid with salting for new goblins-processes.
	now := time.Now()
	rand.Seed(now.UnixNano())

	randNum := rand.Intn(MAX-MIN+1) + MIN
	uniquePid := strconv.Itoa(randNum + now.Nanosecond())

	// Spawn a new goblin-process for incoming websocket.
	pid := engine.Spawn(newGoblin, uniquePid)

	// Create a channel to  break  the websocket handling scope.
	quitCh := make(chan struct{})

	// Send datas the created goblin-process.
	// Than goblin-process will hold (the websocket connection) and (the quitCh pointer).
	engine.Send(pid, &initValues{
		ws:     ws,
		quitCh: &quitCh,
	})

	return pid, &quitCh
}
