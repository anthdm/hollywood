package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/anthdm/hollywood/actor"
)

type Server struct {
	ctx      *actor.Context
	sessions map[string]*actor.PID
}

func NewServer() actor.Receiver {
	return &Server{
		sessions: make(map[string]*actor.PID),
	}
}

func (s *Server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		log.Println("Server started on port 8080")
		s.serve()
		s.ctx = ctx
		_ = msg
	case Message:
		s.broadcast(ctx.Sender(), msg)
	default:
		fmt.Printf("Server received %v\n", msg)
	}
}

func (s *Server) serve() {
	go func() {
		http.HandleFunc("/ws", s.handleWebsocket)
		http.ListenAndServe(":8080", nil)
	}()
}

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New connection")
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	username := r.URL.Query().Get("username")
	pid := s.ctx.SpawnChild(NewUser(username, conn, s.ctx.PID()), username)

	s.sessions[pid.GetID()] = pid
}

func (s *Server) broadcast(sender *actor.PID, msg Message) {
	for _, pid := range s.sessions {
		if !pid.Equals(sender) {
			s.ctx.Send(pid, msg)
		}
	}
}

type Message struct {
	Content string `json:"content"`
	Owner   string `json:"owner"`
}

type User struct {
	conn      *websocket.Conn
	ctx       *actor.Context
	serverPid *actor.PID
	Name      string
}

func NewUser(name string, conn *websocket.Conn, serverPid *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &User{
			Name:      name,
			conn:      conn,
			serverPid: serverPid,
		}
	}
}

func (u *User) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		u.ctx = ctx
		go u.listen()
	case Message:
		u.send(&msg)
	case actor.Stopped:
		_ = msg
		u.conn.Close()
	default:
		fmt.Printf("%s received %v\n", u.Name, msg)
	}
}

func (u *User) listen() {
	var msg Message
	for {
		if err := u.conn.ReadJSON(&msg); err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		msg.Owner = u.Name

		go u.handleMessage(msg)
	}
}

func (u *User) handleMessage(msg Message) {
	switch msg.Content {
	case "exit": // Send exit message to stop the actor and close the websocket connection
		u.ctx.Engine().Poison(u.ctx.PID())
	default:
		// Note that this is the server pid, so it will broadcast the message
		u.ctx.Send(u.serverPid, msg)
	}
}

func (u *User) send(msg *Message) {
	if err := u.conn.WriteJSON(msg); err != nil {
		fmt.Printf("Error writing message: %v\n", err)
		return
	}
}

func main() {
	engine := actor.NewEngine()
	engine.Spawn(NewServer, "server")

	select {}
}
