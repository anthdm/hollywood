package actor

import (
	fmt "fmt"
	"sync"
	"testing"
	"time"
)

type testProc struct {
	inbox Inboxer
}

func (testProc) Start()    {}
func (testProc) PID() *PID { return NewPID("foo", "foo") }
func (t *testProc) Send(_ *PID, msg any, _ *PID) {
	t.inbox.Send(Envelope{Msg: msg})
}

func (t *testProc) Invoke(msgs []Envelope) {
	fmt.Println("got", msgs)
}

func (testProc) Shutdown(_ *sync.WaitGroup) {}

func TestInbox(t *testing.T) {
	inbox := NewInbox(1024)
	inbox.Start(&testProc{inbox})
	inbox.Send(Envelope{Msg: "1"})
	inbox.Send(Envelope{Msg: "2"})
	inbox.Send(Envelope{Msg: "3"})
	inbox.Send(Envelope{Msg: "4"})
	inbox.Send(Envelope{Msg: "5"})
	time.Sleep(time.Second * 2)
}
