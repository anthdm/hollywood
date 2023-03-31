package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

var (
	engine         *actor.Engine
	hodorProcessId *actor.PID
)

// It is storing goblin-process and coressponding websocket identities.
// Also it broadcasts messages to all goblin-processes.
func (f *hodorStorage) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[HODOR] storage has started, id:", ctx.PID().ID)

	//----------------------------
	//-----CAUSES RACE COND.------
	//----------------------------
	// case *letterToHodor:
	// 	fmt.Println("[HODOR] message has received from:", ctx.PID())
	//----------------------------
	//-----CAUSES RACE COND.------
	//----------------------------

	// Delete the incoming websocket value.
	case *deleteFromHodor:

		msg.deleteWsCh <- msg.ws

		go func(f *hodorStorage, msg *deleteFromHodor) {
			for wsocket := range msg.deleteWsCh {
				delete(f.storage, wsocket)
				close(msg.deleteWsCh)
				break
			}
		}(f, msg)

	// Add  a new websocket and  goblin-process  pair.
	case *addToHodor:

		// Prepare  a new map  to  send over  channel.
		newMap := make(map[*websocket.Conn]*actor.PID)
		newMap[msg.ws] = msg.pid

		msg.addWsCh <- addToHodor{
			ws:  msg.ws,
			pid: msg.pid,
		}

		go func(f *hodorStorage, msg *addToHodor) {
			for wsMap := range msg.addWsCh {
				f.storage[wsMap.ws] = wsMap.pid
				close(msg.addWsCh)
				break
			}
		}(f, msg)

	case *broadcastMessage:
		go func(msg *broadcastMessage) {
			for _, pid := range f.storage {
				// Send data to its own goblin-processor for send message its own websocket.
				// So goblin-process will write own message to own websocket.
				engine.Send(pid, msg)
			}
		}(msg)

	}
}

// Goblin-processes corresponding to websockets live here..
func (f *websocketGoblin) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("[WEBSOCKET] foo has started:", ctx.PID().ID)

	case *initValues:
		// If exist, make sure you have one websocket.
		if f.exist {
			break
		}

		// Change states.
		f.ws = msg.ws
		f.exist = true
		f.quitCh = msg.quitCh

		// Add websocket and corresponding goblin-processId pair to hodor-storage.
		engine.Send(hodorProcessId, &addToHodor{
			ws:      msg.ws,
			pid:     ctx.PID(),
			addWsCh: make(chan addToHodor),
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

				// If you want, show that in console.
				fmt.Println("message.....:", message)

				// Send message to  the HODOR  for broadcasting.
				// HODOR will broadcast messages to all goblins.
				engine.Send(hodorProcessId, &broadcastMessage{
					data: message,
				})

				// If not exist, break the loop.
				// No need to read from the client anymore.
				if !f.exist {
					break
				}
			}

		}(ourWs)

	// Broadcast messages  to all goblin-processes.
	case *broadcastMessage:
		_, err := f.ws.Write([]byte(msg.data))
		if err != nil {
			engine.Send(ctx.PID(), &closeWebSocket{
				ws: f.ws,
			})
		}

	// Close the specific goblin-process and websocket pair.
	case *closeWebSocket:
		// Changing  the  f.exist  value   to false causing-
		// the "reader loop" to break. So we don't need read anymore.
		f.exist = false

		// Close the channel. So that will break the websocket scope.
		*f.quitCh <- struct{}{}

		// Delete the websocket and goblin-process pair in the hodor-storage.
		engine.Send(hodorProcessId, &deleteFromHodor{
			ws:         msg.ws,
			deleteWsCh: make(chan *websocket.Conn),
		})

		// Poison the goblin-process.
		wg := &sync.WaitGroup{}
		ctx.Engine().Poison(ctx.PID(), wg)

	case actor.Stopped:
		fmt.Println("goblin-process is stopped:", ctx.PID().ID)

	}
}

func newHodor() actor.Receiver {
	return &hodorStorage{
		storage: make(map[*websocket.Conn]*actor.PID),
	}
}

func newGoblin() actor.Receiver {
	return &websocketGoblin{
		ws:    &websocket.Conn{},
		exist: false,
	}
}

func main() {

	// Create a new  engine.
	// Ash   nazg   durbatulûk ,  ash  nazg  gimbatul,
	// ash nazg thrakatulûk agh burzum-ishi krimpatul.
	engine = actor.NewEngine()

	// Spawn a HODOR(holder-of-the-storage).
	hodorProcessId = engine.Spawn(newHodor, "HODOR_STORAGE")

	// Handle websocket connections. Then we will create  a  new process-
	// corresponding  websocket. Goblin-process  is  carrying websockets.
	http.Handle("/ws", websocket.Handler(HandleFunc(GenerateProcessForWs)))
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}

}
