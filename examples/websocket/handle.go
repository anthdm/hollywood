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
		// If we can't write to the socket later and we get-
		// a "write error", we break this (websocket) scope.
		//
		// Generating a new process  for websocket.
		// Getting a quitChanel to break websocket scope.
		pid, quitCh := f(c)
		fmt.Println("new websocket process is spawned: ", pid.ID)

		for {
			// Waiting for  the "break call"  from processes.
			<-*quitCh

			// Kill the websocket.
			break
		}
	}
}

func GenerateProcessForWs(ws *websocket.Conn) (*actor.PID, *chan struct{}) {
	// Create unique pid with salting for new process.
	now := time.Now()
	rand.Seed(now.UnixNano())
	randNum := rand.Intn(max-min+1) + min
	uniquePid := strconv.Itoa(randNum + now.Nanosecond())

	// Spawn a new process for the new socket.
	pid := engine.Spawn(newGoblin, uniquePid)

	// Create a channel to  break  the socket.
	quitCh := make(chan struct{})

	// Send  datas   which  is   init  values.
	engine.Send(pid, &initValues{
		ws:     ws,
		quitCh: &quitCh,
	})

	return pid, &quitCh
}
