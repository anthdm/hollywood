package cluster

import mapset "github.com/deckarep/golang-set/v2"

type KindLookup struct {
	kinds map[string]mapset.Set[ActiveKind]
}

func NewKindLookup() *KindLookup {
	return &KindLookup{
		kinds: make(map[string]mapset.Set[ActiveKind]),
	}
}

func (l *KindLookup) Add(kind ActiveKind) {
	entry, ok := l.kinds[kind.cid.Kind]
	if ok {
		entry.Add(kind)
		return
	}
	l.kinds[kind.cid.Kind] = mapset.NewSet[ActiveKind]()
	l.kinds[kind.cid.Kind].Add(kind)
}

func (l *KindLookup) Get(kind string) []ActiveKind {
	if kinds, ok := l.kinds[kind]; ok {
		return kinds.ToSlice()
	}
	return []ActiveKind{}
}

func (l *KindLookup) Remove(name string, kind ActiveKind) {

}

func (l *KindLookup) Has(kind ActiveKind) bool {
	kinds, ok := l.kinds[kind.cid.Kind]
	if !ok {
		return false
	}
	found := false
	kinds.Each(func(akind ActiveKind) bool {
		if kind.cid.Equals(akind.cid) && kind.pid.Equals(akind.pid) {
			found = true
			return false
		}
		return true
	})
	return found
}
