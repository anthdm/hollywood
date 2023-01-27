package safemap

import "sync"

type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func New[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

func (s *SafeMap[K, V]) Set(k K, v V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}

func (s *SafeMap[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[k]
	return val, ok
}

func (s *SafeMap[K, V]) Delete(k K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, k)
}

func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *SafeMap[K, V]) ForEach(f func(K, V)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, val := range s.data {
		f(key, val)
	}
}
