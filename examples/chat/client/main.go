package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/remote"
)

type client struct {
	username  string
	serverPID *actor.PID
	logger    *slog.Logger
}

func newClient(username string, serverPID *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &client{
			username:  username,
			serverPID: serverPID,
			logger:    slog.Default(),
		}
	}
}

func (c *client) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case *types.Message:
		fmt.Printf("%s: %s\n", msg.Username, msg.Msg)
	case actor.Started:
		ctx.Send(c.serverPID, &types.Connect{
			Username: c.username,
		})
	case actor.Stopped:
		c.logger.Info("client stopped")
	}
}

func main() {
	var (
		listenAt  = flag.String("listen", "", "specify address to listen to, will pick a random port if not specified")
		connectTo = flag.String("connect", "127.0.0.1:4000", "the address of the server to connect to")
		username  = flag.String("username", os.Getenv("USER"), "")
	)
	flag.Parse()
	if *listenAt == "" {
		*listenAt = fmt.Sprintf("127.0.0.1:%d", rand.Int31n(50000)+10000)
	}
	rem := remote.New(*listenAt, remote.NewConfig())
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(rem))
	if err != nil {
		slog.Error("failed to create engine", "err", err)
		os.Exit(1)
	}

	var (
		// the process ID of the server
		serverPID = actor.NewPID(*connectTo, "server/primary")
		// Spawn our client receiver
		clientPID = e.Spawn(newClient(*username, serverPID), "client", actor.WithID(*username))
		scanner   = bufio.NewScanner(os.Stdin)
	)
	fmt.Println("Type 'quit' and press return to exit.")
	for scanner.Scan() {
		msg := &types.Message{
			Msg:      scanner.Text(),
			Username: *username,
		}
		// We use SendWithSender here so the server knows who
		// is sending the message.
		if msg.Msg == "quit" {
			break
		}
		e.SendWithSender(serverPID, msg, clientPID)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("failed to read message from stdin", "err", err)
	}

	// When breaked out of the loop on error let the server know
	// we need to disconnect.
	e.SendWithSender(serverPID, &types.Disconnect{}, clientPID)
	e.Poison(clientPID).Wait()
	slog.Info("client disconnected")
}
