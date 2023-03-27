package main

import (
	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

// We need faithful reliable friend who is -
// HODOR(holder-of-the-storage) to bridle goblins.
type hodorStorage struct {
	storage map[*websocket.Conn]*actor.PID
}

// We have goblin(holder-of-the-websocket)-process.
type websocketGoblin struct {
	ws     *websocket.Conn
	exist  bool
	quitCh *chan struct{}
}

// We can easily kill goblin-process with chan struct.
type closeWebSocket struct {
	ws *websocket.Conn
}

// We can create or delete goblin data in -
// hodor-storage.
type letterToHodor struct {
	pid  *actor.PID
	ws   *websocket.Conn
	drop bool
}

// We can pass the message of a goblin-pprocess.
type broadcastMessage struct {
	data string
}

// Goblins will need armor and spears.
type initValues struct {
	ws     *websocket.Conn
	quitCh *chan struct{}
}
