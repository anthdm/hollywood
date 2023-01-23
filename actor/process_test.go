package actor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var restarts = 0

type testActor struct{}

func newTestActor() Receiver { return &testActor{} }
func (*testActor) Receive(ctx *Context) {
	switch msg := ctx.Message().(type) {
	case Started:
		if restarts < 1 {
			restarts++
			panic("d")
		}
	default:
		fmt.Println(msg)
	}
}

func TestProcess(t *testing.T) {
	e := NewEngine()
	proc := NewProcess(e, DefaultProducerConfig(newTestActor))
	proc.Name = "ddd"
	pid := proc.start()
	assert.NotNil(t, pid)
	time.Sleep(time.Second * 1)
	e.Send(pid, "dd")

	time.Sleep(time.Second * 10)
}
