package main

import (
	"fmt"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

type handleWithPid func(ws *websocket.Conn) (*actor.PID, *chan struct{})

func HandleFunc(f handleWithPid) websocket.Handler {
	return func(c *websocket.Conn) {
		defer fmt.Println("web socket is deleted")

		// We get QuitChannel. If we can't write later and we get a write error, we break this scope (websocket).
		pid, quitCh := f(c)
		fmt.Println("new websocket is spawned: ", pid.ID)

		for {
			// Waiting for break call.
			<-*quitCh
			engine.Send(pid, &closeWsMsg{
				ws: c,
			})

			// Kill the websocket.
			break
		}
	}
}

func GenerateProcessForWs(ws *websocket.Conn) (*actor.PID, *chan struct{}) {

	// Spawn new pid for new socket.
	pid := engine.Spawn(webSocketFoo, ws.RemoteAddr().String())

	// Create a channel to break the socket.
	quitCh := make(chan struct{})

	// Send datas which is init values.
	engine.Send(pid, &setWsVal{
		ws:     ws,
		quitCh: &quitCh,
	})

	return pid, &quitCh
}
