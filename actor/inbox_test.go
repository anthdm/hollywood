package actor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDdkjfkdjfkfd(t *testing.T) {
	x := 78225
	size := 1024 * 64
	mask := size - 1
	fmt.Println(x & mask)
	// 12689
}

func TestXxx(t *testing.T) {
	e := NewEngine()
	i := 0

	pid := e.SpawnFunc(func(c *Context) {
		if msg, ok := c.Message().(int); ok {
			// fmt.Println("got", msg)
			require.Equal(t, msg, i)
			i++
		}
	}, "test", WithInboxSize(1024*64))

	for i := 0; i < 1000000; i++ {
		e.Send(pid, i)
	}
	fmt.Println("done")
	time.Sleep(time.Second)
}
