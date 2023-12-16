package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveKind(t *testing.T) {
	lookup := NewKindLookup()
	kind := ActiveKind{
		cid: &CID{Kind: "foo", ID: "1"},
	}
	lookup.Add(kind)
	assert.Equal(t, lookup.Get("foo")[0], kind)
	lookup.Remove(kind)

	assert.Len(t, lookup.Get("foo"), 0)
	assert.Empty(t, lookup.kinds)
}
