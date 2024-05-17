package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixMapBuilder[T any] struct {
	root node[T]
}

func prefixFromLabel(l label) netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], l.value.hi)
	bePutUint64(a16[8:], l.value.lo)
	return netip.PrefixFrom(netip.AddrFrom16(a16), int(l.len))
}

func (m *PrefixMapBuilder[T]) Set(prefix netip.Prefix, value T) error {
	if !prefix.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", prefix)
	}
	m.root.set(labelFromPrefix(prefix), value)
	return nil
}

type PrefixMap[T any] struct {
	root node[T]
}

func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{root: *m.root.copy()}
}

// Get returns the value associated with the exact prefix provided, if any.
func (m *PrefixMap[T]) Get(prefix netip.Prefix) (T, bool) {
	return m.root.get(labelFromPrefix(prefix))
}

// GetDescendants returns all descendants of prefix found in the map (including
// prefix, if it has a value) as a map of prefixes to values.
func (m *PrefixMap[T]) GetDescendants(prefix netip.Prefix) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.root.walkDescendants(labelFromPrefix(prefix), func(l label, n *node[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

// GetAncestors returns all ancestors of prefix found in the map (including
// prefix, if it has a value) as a map of prefixes to values.
func (m *PrefixMap[T]) GetAncestors(prefix netip.Prefix) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.root.walkAncestors(labelFromPrefix(prefix), func(l label, n *node[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}
