package actor

import (
	"fmt"
	"github.com/anthdm/hollywood/log"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

type testActor struct {
}
type testMessage struct {
	data string
}

func newTestActor() Receiver {
	return testActor{}
}
func (t testActor) Receive(ctx *Context) {
	// do nothing
}

func TestDeadLetterDefault(t *testing.T) {
	lh := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelDebug)
	e := NewEngine(EngineOptLogger(log.NewLogger("[engine]", lh)))
	a1 := e.Spawn(newTestActor, "a1")
	assert.NotNil(t, a1)
	dl := e.Registry.getByID("deadletter")
	assert.NotNil(t, dl) // should be registered by default
	// kill a1 actor.
	e.Poison(a1).Wait() // poison the a1 actor
	// should be in deadletter
	fmt.Println("==== sending message via a1 to deadletter ====")
	e.Send(a1, testMessage{"bar"})
	time.Sleep(time.Millisecond * 50)
	resp, err := e.Request(dl.PID(), &DeadLetterFetch{Flush: true}, time.Millisecond*1000).Result()
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	resp2, ok := resp.([]*DeadLetterEvent)
	assert.True(t, ok)
	assert.Equal(t, 1, len(resp2))

}
