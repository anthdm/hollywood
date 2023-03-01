package ggq

import (
	"fmt"
	"testing"
)

type consumer[T any] struct{}

func (c *consumer[T]) Consume(t []T) {
	fmt.Println("len:", len(t))
}

func TestXddd(t *testing.T) {
	q := New[int](1024, &consumer[int]{})

	go func() {
		for i := 0; i < 100000; i++ {
			q.Write(i)
		}
		q.Close()
	}()

	q.ReadN()
}

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
