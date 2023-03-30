package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/websocket"
)

func TestHandleWebsocket(t *testing.T) {

	// Create a new engine to spawn Hodor-Storage..
	engine = actor.NewEngine()
	hodorProcessId = engine.Spawn(newHodor, "HODOR_STORAGE")

	s := httptest.NewServer(HandleFunc(GenerateProcessForWs))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	println(u)

	// Connect to the server
	ws, err := websocket.Dial(u, "", u)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	go func(ws *websocket.Conn) {
		for {
			buf := make([]byte, 1024)
			_, err = ws.Read(buf)
			if err != nil {
				continue
			}
			// println(string(buf))
		}
	}(ws)

	for i := 0; i < 10; i++ {
		_, err := ws.Write([]byte("Some text."))
		if err != nil {
			t.Fatalf("WS Write Error: %v", err)
		}
	}

}
