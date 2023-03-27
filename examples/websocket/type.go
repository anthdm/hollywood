package main

import (
	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

type wsFoo struct {
	ws     *websocket.Conn
	exist  bool
	quitCh *chan struct{}
}

type sendStorageMsg struct {
	pid  *actor.PID
	ws   *websocket.Conn
	drop bool
}

type broadcastMsg struct {
	data string
}

type wsPidStore struct {
	storage map[*websocket.Conn]*actor.PID
}

type setWsVal struct {
	ws     *websocket.Conn
	quitCh *chan struct{}
}

type closeWsMsg struct {
	ws *websocket.Conn
}
