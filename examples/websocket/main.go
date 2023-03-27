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

// It is storing ProcessId and WebSocket identities.
// Also it broadcasts messages to all processes.
func (f *wsPidStore) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[wsPidStore] storage has started")

	case *sendStorageMsg:
		fmt.Println("[wsPidStore] message has received", msg.ws)
		// Delete incoming socket value.
		if msg.drop {
			delete(f.storage, msg.ws)
		} else {
			f.storage[msg.ws] = msg.pid
		}

	case *broadcastMsg:
		go func(msg *broadcastMsg) {
			for _, pid := range f.storage {
				engine.Send(pid, msg)
			}
		}(msg)

	}
}

// Accepted (web)sockets.
func (f *wsFoo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[WEBSOCKET] foo has started")

	case *setWsVal:
		// If exist, make sure you have one socket.
		if f.exist {
			break
		}
		// Change states.
		f.ws = msg.ws
		f.exist = true
		f.quitCh = msg.quitCh

		// Add socket and procesId pair to storage.
		engine.Send(storageProcessId, &sendStorageMsg{
			pid:  ctx.PID(),
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
				msgStr := string(msg)

				message := fmt.Sprintf("%s:%s", ourWs.RemoteAddr().String(), msgStr)
				// If u want show in console.
				fmt.Println("message:", message)

				// Send message to storage for broadcasting.
				// Storage will broadcast messages to all pids.
				engine.Send(storageProcessId, &broadcastMsg{
					data: message,
				})

				// If not exist, break the loop. No need to read from the client anymore.
				if !f.exist {
					break
				}
			}

		}(ourWs)

	// Broadcast messages to all pids.
	case *broadcastMsg:
		_, err := f.ws.Write([]byte(msg.data))
		if err != nil {
			engine.Send(ctx.PID(), &closeWsMsg{
				ws: f.ws,
			})
		}

	// Close the specific socket and pid pair.
	case *closeWsMsg:
		// Changing the existing state to false causes the reader loop to break.
		// We don't need read anymore.
		f.exist = false

		// Close the channel. Break the (web)socket whic is in handler func.
		*f.quitCh <- struct{}{}

		// Delete the web socket in the repository.
		engine.Send(storageProcessId, &sendStorageMsg{
			ws:   f.ws,
			drop: true,
		})

		fmt.Println("socket processor is deleted.")
		// Break a process.
		return
	}
}

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
