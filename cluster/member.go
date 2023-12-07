package cluster

import "github.com/anthdm/hollywood/actor"

// MemberToPID creates a new PID from the member info.
func MemberToPID(m *Member) *actor.PID {
	return actor.NewPID(m.Host, "cluster/"+m.ID)
}

type Map[K comparable, V any] struct {
	data map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data: make(map[K]V),
	}
}

func (m *Map[K, V]) Add(k K, v V) {
	m.data[k] = v
}

func (m *Map[K, V]) Delete(k K) {
	delete(m.data, k)
}

func (m *Map[K, V]) Contains(k K) bool {
	_, ok := m.data[k]
	return ok
}

func (m *Map[K, V]) Len() int {
	return len(m.data)
}

func (m *Map[K, V]) Slice() []V {
	s := make([]V, m.Len())
	i := 0
	for _, v := range m.data {
		s[i] = v
		i++
	}
	return s
}
