package actor

import (
	sync "sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventStream(t *testing.T) {
	e := NewEngine()
	wg := sync.WaitGroup{}
	subs := []*EventSub{}
	var mu sync.RWMutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			sub := e.EventStream.Subscribe(func(event any) {
				s, ok := event.(string)
				assert.True(t, ok)
				assert.Equal(t, "foo", s)
			})

			e.EventStream.Publish("foo")
			mu.Lock()
			subs = append(subs, sub)
			mu.Unlock()
			wg.Done()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 10, e.EventStream.Len())

	for _, sub := range subs {
		e.EventStream.Unsubscribe(sub)
	}

	assert.Equal(t, 0, e.EventStream.Len())
}
