package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixMapBuilder[T any] struct {
	tree node[T]
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
	m.tree.set(labelFromPrefix(prefix), value)
	return nil
}

type PrefixMap[T any] struct {
	tree node[T]
}

func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{tree: *m.tree.copy()}
}

// Get returns the value associated with the exact prefix provided, if any.
func (m *PrefixMap[T]) Get(prefix netip.Prefix) (T, bool) {
	return m.tree.get(labelFromPrefix(prefix))
}

func (m *PrefixMap[T]) descendentsOf(
	prefix netip.Prefix,
	strict bool,
) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walkDescendants(labelFromPrefix(prefix), strict, func(l label, n *node[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

// DescendantsOf returns all descendants of prefix (including prefix itself,
// if it has a value) as a map of prefixes to values.
func (m *PrefixMap[T]) DescendantsOf(prefix netip.Prefix) map[netip.Prefix]T {
	return m.descendentsOf(prefix, false)
}

// StrictDescendantsOf returns all descendants of prefix as a map of prefixes to
// values.
func (m *PrefixMap[T]) StrictDescendantsOf(prefix netip.Prefix) map[netip.Prefix]T {
	return m.descendentsOf(prefix, true)
}

// AncestorsOf returns all ancestors of prefix as a map of prefixes to values.
func (m *PrefixMap[T]) ancestorsOf(
	prefix netip.Prefix,
	strict bool,
) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walkAncestors(labelFromPrefix(prefix), false, func(l label, n *node[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

// AncestorsOf returns all ancestors of prefix (including prefix itself, if it
// has a value) as a map of prefixes to values.
func (m *PrefixMap[T]) AncestorsOf(prefix netip.Prefix) map[netip.Prefix]T {
	return m.ancestorsOf(prefix, false)
}

// StrictAncestorsOf returns all ancestors of prefix as a map of prefixes to values.
func (m *PrefixMap[T]) StrictAncestorsOf(prefix netip.Prefix) map[netip.Prefix]T {
	return m.ancestorsOf(prefix, true)
}

func (m *PrefixMap[T]) rootOf(
	prefix netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	label, val, ok := m.tree.rootOf(labelFromPrefix(prefix), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromLabel(label), val, true
}

// RootOf returns the shortest-prefix ancestor of the prefix provided, if any.
// The prefix itself is returned if it has no ancestors and has a value.
func (m *PrefixMap[T]) RootOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, false)
}

// RootOf returns the shortest-prefix ancestor of the prefix provided, if any.
// If the prefix has no ancestors, RootOf returns zero values and false.
func (m *PrefixMap[T]) StrictRootOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, true)
}

func (m *PrefixMap[T]) parentOf(
	prefix netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	label, val, ok := m.tree.parentOf(labelFromPrefix(prefix), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromLabel(label), val, true
}

// ParentOf returns the longest-prefix ancestor of the prefix provided, if any.
// If the prefix has no ancestors, ParentOf returns zero values and false.
func (m *PrefixMap[T]) ParentOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, false)
}

// StrictParentOf returns the longest-prefix ancestor of the prefix provided, if any.
// If the prefix has no ancestors, StrictParentOf returns zero values and false.
func (m *PrefixMap[T]) StrictParentOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, true)
}
