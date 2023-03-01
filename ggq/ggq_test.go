package ggq

import (
	"testing"
)

type consumer[T any] struct{}

func (c *consumer[T]) Consume(t []T) {
	// fmt.Println(len(t))
}

func TestSingleMessageNotConsuming(t *testing.T) {
	q := New[int](1024, &consumer[int]{})

	go func() {
		for i := 0; i < 1; i++ {
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
