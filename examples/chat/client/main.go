package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/remote"
)

type client struct {
	username  string
	serverPID *actor.PID
}

func newClient(username string, serverPID *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &client{
			username:  username,
			serverPID: serverPID,
		}
	}
}

func (c *client) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case *types.Message:
		fmt.Printf("username: %s :: %s\n", msg.Username, msg.Msg)
	case actor.Started:
		ctx.Send(c.serverPID, &types.Connect{
			Username: c.username,
		})
	case actor.Stopped:
	}
}

func main() {
	var (
		listenAt  = flag.String("listen", "127.0.0.1:3000", "")
		connectTo = flag.String("connect", "127.0.0.1:4000", "")
		username  = flag.String("username", "", "")
	)
	flag.Parse()

	e := actor.NewEngine()
	rem := remote.New(e, remote.Config{
		ListenAddr: *listenAt,
	})
	e.WithRemote(rem)

	var (
		// the process ID of the server
		serverPID = actor.NewPID(*connectTo, "server")
		// Spawn our client receiver
		clientPID = e.Spawn(newClient(*username, serverPID), "client")
		scanner   = bufio.NewScanner(os.Stdin)
	)
	for scanner.Scan() {
		msg := &types.Message{
			Msg:      scanner.Text(),
			Username: *username,
		}
		// We use SendWithSender here so the server knows who
		// is sending the message.
		e.SendWithSender(serverPID, msg, clientPID)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("failed to read message from stdin", "err", err)
	}

	// When breaked out of the loop on error let the server know
	// we need to disconnect.
	e.SendWithSender(serverPID, &types.Disconnect{}, clientPID)
}
