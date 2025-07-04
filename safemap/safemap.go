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
	for k, v := range s.data {
		f(k, v)
	}
}

func (s *SafeMap[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

func (s *SafeMap[K, V]) Values() []V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]V, 0, len(s.data))
	for _, v := range s.data {
		values = append(values, v)
	}
	return values
}
