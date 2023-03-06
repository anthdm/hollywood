package ggq

import (
	"fmt"
	"testing"
)

type consumer[T any] struct{}

func (c *consumer[T]) Consume(t []T) {
	fmt.Println(t)
}

func TestSingleMessageNotConsuming(t *testing.T) {
	q := New[int](1024, &consumer[int]{})

	q.cond.Signal()
	for i := 0; i < 10; i++ {
		q.Write(i)
	}
	q.Close()

	q.ReadN()
}

// 35 ns/op		0 B/op		0 allocs/op
func BenchmarkReadN(b *testing.B) {
	q := New[int](1024*12, &consumer[int]{})
	go func() {
		for i := 0; i < b.N; i++ {
			q.Write(i)
		}
		q.Close()
	}()

	q.ReadN()
}

func BenchmarkRead(b *testing.B) {
	q := New[int](1024*12, &consumer[int]{})

	go func() {
		for i := 0; i < b.N; i++ {
			q.Write(i)
		}
		q.Close()
	}()

	for {
		_, closed := q.Read()
		if closed {
			break
		}
	}
}
