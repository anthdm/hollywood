package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

var (
	engine           *actor.Engine
	storageProcessId *actor.PID
)

func webSocketStorage() actor.Receiver {
	return &wsPidStore{
		storage: make(map[*websocket.Conn]*actor.PID),
	}
}

func webSocketFoo() actor.Receiver {
	return &wsFoo{
		ws:    &websocket.Conn{},
		exist: false,
	}
}

// It is just one process. So only store socket data.
func (f *wsPidStore) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[wsPidStore] storage has started")
	case *sendStorageMsg:
		fmt.Println("[wsPidStore] message has received", msg.ws)
		// Delete incoming socket value.
		if !msg.drop {
			f.storage[msg.ws] = msg.pid
		} else {
			delete(f.storage, msg.ws)
		}
	}
}

// Accepted (web)sockets.
func (f *wsFoo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[WEBSOCKET] foo has started")

	case *setWsVal:
		// If exist, no need to assign.
		if f.exist {
			break
		}
		// Change state.
		f.ws = msg.ws
		f.exist = true

		// Send values to storage.
		engine.Send(storageProcessId, &sendStorageMsg{
			ws:   f.ws,
			drop: false,
		})

		ourWs := msg.ws
		// Generate a reading loop. Read incoming messages.
		go func(ourWs *websocket.Conn) {
			buf := make([]byte, 1024)
			for {
				n, err := ourWs.Read(buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					fmt.Println("read error: ", err)
					continue
				}
				msg := buf[:n]
				fmt.Println("message:", string(msg)) // convert value to string because we are human.
				// If not exist, break the loop. No more need read from client.
				if !f.exist {
					break
				}
			}

		}(ourWs)
	case *closeWsMsg:
		// In handle func, we are checking the client which is alive or not.
		// Then send a *closeWsMsg message  to relevant socket.
		// Relevant socket is this scope.
		//
		// Change exist state to false.
		f.exist = false
		engine.Send(storageProcessId, &sendStorageMsg{
			ws:   f.ws,
			drop: true,
		})

		fmt.Println("socket processor is deleted.")
		// Break a process.
		return
	}
}

func main() {

	// Create new engine.
	engine = actor.NewEngine()

	// Spawn a storage process.
	storageProcessId = engine.Spawn(webSocketStorage, "storer")

	// Handle WebSockets.
	http.Handle("/ws", websocket.Handler(HandleFunc(GenerateProcessForWs)))
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}

}
