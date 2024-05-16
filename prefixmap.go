package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixMapBuilder[T any] struct {
	root node[T]
}

type PrefixMap[T any] struct {
	root node[T]
}

func (m *PrefixMapBuilder[T]) Set(prefix netip.Prefix, value T) error {
	if !prefix.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", prefix)
	}
	m.root.set(labelFromPrefix(prefix), value)
	return nil
}

func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{root: *m.root.copy()}
}

// Get returns the value associated with the exact prefix m, if any.
func (m *PrefixMap[T]) Get(prefix netip.Prefix) (T, bool) {
	return m.root.get(labelFromPrefix(prefix))
}

func prefixFromLabel(l label) netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], l.value.hi)
	bePutUint64(a16[8:], l.value.lo)
	return netip.PrefixFrom(netip.AddrFrom16(a16), int(l.len))
}

// GetDescendants returns all descendants of prefix found in m (including prefix,
// if it has a value) as a map of prefixes to values.
func (m *PrefixMap[T]) GetDescendants(prefix netip.Prefix) map[netip.Prefix]T {
	desc := m.root.getDescendants(labelFromPrefix(prefix))
	res := make(map[netip.Prefix]T, len(desc))
	for l, v := range desc {
		res[prefixFromLabel(l)] = v
	}
	return res
}
