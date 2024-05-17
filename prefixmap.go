package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixMapBuilder[T any] struct {
	tree tree[T]
}

func prefixFromLabel(l label) netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], l.value.hi)
	bePutUint64(a16[8:], l.value.lo)
	return netip.PrefixFrom(netip.AddrFrom16(a16), int(l.len))
}

func (m *PrefixMapBuilder[T]) Set(p netip.Prefix, value T) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.set(labelFromPrefix(p), value)
	return nil
}

func (m *PrefixMapBuilder[T]) Remove(p netip.Prefix) error {
	fmt.Println("Remove")
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.remove(labelFromPrefix(p))
	return nil
}

func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{tree: *m.tree.copy()}
}

type PrefixMap[T any] struct {
	tree tree[T]
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMap[T]) Get(p netip.Prefix) (T, bool) {
	return m.tree.get(labelFromPrefix(p))
}

// Contains returns true if this map includes the exact Prefix provided.
func (m *PrefixMap[T]) Contains(p netip.Prefix) bool {
	return m.tree.contains(labelFromPrefix(p))
}

// Encompasses returns true if this map includes a Prefix which completely
// encompasses the provided Prefix.
func (m *PrefixMap[T]) Encompasses(p netip.Prefix) bool {
	return m.tree.encompasses(labelFromPrefix(p), false)
}

// EncompassesStrict returns true if this map includes a Prefix which
// completely encompasses the provided Prefix.
func (m *PrefixMap[T]) EncompassesStrict(p netip.Prefix) bool {
	return m.tree.encompasses(labelFromPrefix(p), true)
}

// Covers returns true if this map includes a subset of Prefixes that
// completely cover provided Prefix.
func (m *PrefixMap[T]) Covers(p netip.Prefix) bool {
	return false
}

func (m *PrefixMap[T]) descendentsOf(
	p netip.Prefix,
	strict bool,
) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walkDescendants(labelFromPrefix(p), strict, func(l label, n *tree[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

func (m *PrefixMap[T]) ToMap() map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walkDescendants(label{}, false, func(l label, n *tree[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

// DescendantsOf returns all descendants of the provided Prefix (including the
// Prefix itself, if it has a value) as a map of Prefixes to values.
func (m *PrefixMap[T]) DescendantsOf(p netip.Prefix) map[netip.Prefix]T {
	return m.descendentsOf(p, false)
}

// StrictDescendantsOf returns all descendants of the provided Prefix as a map
// of Prefixes to values.
func (m *PrefixMap[T]) StrictDescendantsOf(p netip.Prefix) map[netip.Prefix]T {
	return m.descendentsOf(p, true)
}

// AncestorsOf returns all ancestors of the provided Prefix as a map of
// Prefixes to values.
func (m *PrefixMap[T]) ancestorsOf(
	p netip.Prefix,
	strict bool,
) map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walkAncestors(labelFromPrefix(p), false, func(l label, n *tree[T]) {
		res[prefixFromLabel(l)] = n.value
	})
	return res
}

// AncestorsOf returns all ancestors of the provided Prefix (including the
// Prefix itself, if it has a value) as a map of Prefixes to values.
func (m *PrefixMap[T]) AncestorsOf(p netip.Prefix) map[netip.Prefix]T {
	return m.ancestorsOf(p, false)
}

// StrictAncestorsOf returns all ancestors of the provided Prefix as a map of
// Prefixes to values.
func (m *PrefixMap[T]) StrictAncestorsOf(p netip.Prefix) map[netip.Prefix]T {
	return m.ancestorsOf(p, true)
}

func (m *PrefixMap[T]) rootOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	label, val, ok := m.tree.rootOf(labelFromPrefix(p), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromLabel(label), val, true
}

// RootOf returns the shortest-prefix ancestor of the Prefix provided, if any.
// The Prefix itself is returned if it has no ancestors and has a value.
func (m *PrefixMap[T]) RootOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, false)
}

// RootOf returns the shortest-prefix ancestor of the Prefix provided, if any.
// If the Prefix has no ancestors, RootOf returns zero values and false.
func (m *PrefixMap[T]) StrictRootOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, true)
}

func (m *PrefixMap[T]) parentOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	label, val, ok := m.tree.parentOf(labelFromPrefix(p), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromLabel(label), val, true
}

// ParentOf returns the longest-prefix ancestor of the Prefix provided, if any.
// If the Prefix has no ancestors, ParentOf returns zero values and false.
func (m *PrefixMap[T]) ParentOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, false)
}

// StrictParentOf returns the longest-prefix ancestor of the Prefix provided,
// if any. If the Prefix has no ancestors, StrictParentOf returns zero values
// and false.
func (m *PrefixMap[T]) StrictParentOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, true)
}
