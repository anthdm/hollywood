package safemap

import "sync"

type SafeMap[K comparable, V any] struct {
	data sync.Map
}

func New[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: sync.Map{},
	}
}

func (s *SafeMap[K, V]) Set(k K, v V) {
	s.data.Store(k, v)
}

func (s *SafeMap[K, V]) Get(k K) (V, bool) {
	val, ok := s.data.Load(k)
	var zero V
	if !ok {
		return zero, false
	}
	return val.(V), ok
}

func (s *SafeMap[K, V]) Delete(k K) {
	s.data.Delete(k)
}

func (s *SafeMap[K, V]) Len() int {
	count := 0
	s.data.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (s *SafeMap[K, V]) ForEach(f func(K, V)) {
	s.data.Range(func(key, value interface{}) bool {
		f(key.(K), value.(V))
		return true
	})
}
