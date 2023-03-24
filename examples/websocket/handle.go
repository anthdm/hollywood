package main

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

type handleWithPid func(ws *websocket.Conn) *actor.PID

func HandleFunc(f handleWithPid) websocket.Handler {
	return func(c *websocket.Conn) {
		defer fmt.Println("web socket is deleted")

		pid := f(c)
		fmt.Println("new websocket is spawned: ", pid.ID)

		for {
			time.Sleep(time.Second * 3)
			// Is client alive?
			_, err := c.Write([]byte("[IFEXIST] are u alive?"))
			if err != nil {
				engine.Send(pid, &closeWsMsg{
					ws: c,
				})
				// Kill the websocket.
				break
			}

		}

	}
}

func GenerateProcessForWs(ws *websocket.Conn) *actor.PID {
	pid := engine.Spawn(webSocketFoo, ws.RemoteAddr().String())
	defer engine.Send(pid, &setWsVal{
		pid: pid,
		ws:  ws,
	})
	return pid
}
