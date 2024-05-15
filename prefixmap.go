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

func (m *PrefixMap[T]) Get(prefix netip.Prefix) (T, bool) {
	return m.root.get(labelFromPrefix(prefix))
}

// GetDescendants returns all descendants of prefic found in m as a map of Prefixes to values.
//func (m *PrefixMap[T]) GetDescendants(prefix netip.Prefix) (map[netip.Prefix]T, bool) {
//}
