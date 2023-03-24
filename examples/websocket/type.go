package main

import (
	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

type wsFoo struct {
	ws    *websocket.Conn
	exist bool
}

type sendStorageMsg struct {
	pid  *actor.PID
	ws   *websocket.Conn
	drop bool
}

type wsPidStore struct {
	storage map[*websocket.Conn]*actor.PID
}

type setWsVal struct {
	pid *actor.PID
	ws  *websocket.Conn
}

type closeWsMsg struct {
	ws *websocket.Conn
}
