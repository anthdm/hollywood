package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
)

type handler struct{}

func newHandler() actor.Receiver {
	return &handler{}
}

func (handler) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case []byte:
		fmt.Println("got message to handle:", string(msg))
	}
}

type session struct {
	conn net.Conn
}

func newSession(conn net.Conn) actor.Producer {
	return func() actor.Receiver {
		return &session{
			conn: conn,
		}
	}
}

func (s *session) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
	case actor.Started:
		log.Infow("new connection", log.M{"addr": s.conn.RemoteAddr()})
		go s.readLoop(c)
	case actor.Stopped:
		s.conn.Close()
	}
}

func (s *session) readLoop(c *actor.Context) {
	buf := make([]byte, 1024)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			log.Errorw("conn read error", log.M{"err": err})
			break
		}
		// copy shared buffer, to prevent race conditions.
		msg := make([]byte, n)
		copy(msg, buf[:n])

		// Send to the handler to process to message
		c.Send(c.Parent().Child("handler"), msg)
	}
	// Loop is done due to error or we need to close due to server shutdown.
	c.Send(c.Parent(), &connRem{pid: c.PID()})
}

type connAdd struct {
	pid  *actor.PID
	conn net.Conn
}

type connRem struct {
	pid *actor.PID
}

type server struct {
	listenAddr string
	ln         net.Listener
	sessions   map[*actor.PID]net.Conn
}

func newServer(listenAddr string) actor.Producer {
	return func() actor.Receiver {
		return &server{
			listenAddr: listenAddr,
			sessions:   make(map[*actor.PID]net.Conn),
		}
	}
}

func (s *server) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Initialized:
		ln, err := net.Listen("tcp", s.listenAddr)
		if err != nil {
			panic(err)
		}
		s.ln = ln
		// start the handler that will handle the incomming messages from clients/sessions.
		c.SpawnChild(newHandler, "handler")
	case actor.Started:
		log.Infow("server started", log.M{"addr": s.listenAddr})
		go s.acceptLoop(c)
	case actor.Stopped:
		// on stop all the childs sessions will automatically get the stop
		// message and close all their underlying connection.
	case *connAdd:
		log.Tracew("added new connection to my map", log.M{"addr": msg.conn.RemoteAddr(), "pid": msg.pid})
		s.sessions[msg.pid] = msg.conn
	case *connRem:
		log.Tracew("removed connection from my map", log.M{"pid": msg.pid})
		delete(s.sessions, msg.pid)
	}
}

func (s *server) acceptLoop(c *actor.Context) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Errorw("accept error", log.M{"err": err})
			break
		}
		pid := c.SpawnChild(newSession(conn), "session", actor.WithTags(conn.RemoteAddr().String()))
		c.Send(c.PID(), &connAdd{
			pid:  pid,
			conn: conn,
		})
	}
}

func main() {
	listenAddr := flag.String("listenaddr", ":6000", "listen address of the TCP server")

	e := actor.NewEngine()
	serverPID := e.Spawn(newServer(*listenAddr), "server")

	// Gracefully shutdown the server and its current connections.
	defer func() {
		e.Poison(serverPID)
		time.Sleep(time.Second)
	}()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	<-sigch
}
