package netipmap

import (
	"fmt"
	"net/netip"
)

// PrefixMapBuilder builds an immutable PrefixMap.
//
// The zero value is a valid value representing an empty PrefixMap.
//
// Call PrefixMap to obtain an immutable PrefixMap from a PrefixMapBuilder.
type PrefixMapBuilder[T any] struct {
	tree tree[T]
}

// Set associates the provided value with the provided Prefix.
func (m *PrefixMapBuilder[T]) Set(p netip.Prefix, value T) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.set(keyFromPrefix(p), value)
	return nil
}

// Remove removes the provided Prefix from m.
func (m *PrefixMapBuilder[T]) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.remove(keyFromPrefix(p))
	return nil
}

// RemoveDescendants removes all Prefixes from m that are encompassed by the
// provided Prefix, including the Prefix itself.
func (m *PrefixMapBuilder[T]) RemoveDescendants(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.removeDescendants(keyFromPrefix(p), false)
	return nil
}

// RemoveDescendantsStrict removes all Prefixes from m that are encompassed by
// the provided Prefix. The provided Prefix itself is not removed.
func (m *PrefixMapBuilder[T]) RemoveDescendantsStrict(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.removeDescendants(keyFromPrefix(p), true)
	return nil
}

// Filter removes all Prefixes from m that are not encompassed by pm.
func (m *PrefixMapBuilder[T]) Filter(pm *PrefixMap[T]) {
	m.tree.filter(pm.tree)
}

// PrefixMap returns an immutable PrefixMap representing the current state of m.
//
// The builder remains usable after calling PrefixMap.
func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.copy()}
}

// PrefixMap is a map of netip.Prefix to T.
//
// Use PrefixMapBuilder to construct PrefixMaps.
type PrefixMap[T any] struct {
	tree tree[T]
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMap[T]) Get(p netip.Prefix) (T, bool) {
	return m.tree.get(keyFromPrefix(p))
}

// Contains returns true if this map includes the exact Prefix provided.
func (m *PrefixMap[T]) Contains(p netip.Prefix) bool {
	return m.tree.contains(keyFromPrefix(p))
}

// Encompasses returns true if this map includes a Prefix which completely
// encompasses the provided Prefix.
func (m *PrefixMap[T]) Encompasses(p netip.Prefix) bool {
	return m.tree.encompasses(keyFromPrefix(p), false)
}

// EncompassesStrict returns true if this map includes a Prefix which
// completely encompasses the provided Prefix. The provided Prefix itself is
// not considered.
func (m *PrefixMap[T]) EncompassesStrict(p netip.Prefix) bool {
	return m.tree.encompasses(keyFromPrefix(p), true)
}

// OverlapsPrefix returns true if this map includes a Prefix which overlaps the
// provided Prefix.
func (m *PrefixMap[T]) OverlapsPrefix(p netip.Prefix) bool {
	return m.tree.overlapsKey(keyFromPrefix(p))
}

func prefixFromKey(b key) netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], b.content.hi)
	bePutUint64(a16[8:], b.content.lo)
	return netip.PrefixFrom(netip.AddrFrom16(a16), int(b.len))
}

func (m *PrefixMap[T]) rootOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	label, val, ok := m.tree.rootOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromKey(label), val, true
}

// RootOf returns the shortest-prefix ancestor of the Prefix provided, if any.
// The Prefix itself is returned if it has no ancestors and has a value.
func (m *PrefixMap[T]) RootOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, false)
}

// RootOf returns the shortest-prefix ancestor of the Prefix provided, if any.
// If the Prefix has no ancestors, RootOf returns zero values and false.
func (m *PrefixMap[T]) RootOfStrict(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.rootOf(p, true)
}

func (m *PrefixMap[T]) parentOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, val T, ok bool) {
	key, val, ok := m.tree.parentOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, val, false
	}
	return prefixFromKey(key), val, true
}

// ParentOf returns the longest-prefix ancestor of the Prefix provided, if any.
// If the Prefix has no ancestors, ParentOf returns zero values and false.
func (m *PrefixMap[T]) ParentOf(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, false)
}

// ParentOfStrict returns the longest-prefix ancestor of the Prefix provided,
// if any. If the Prefix has no ancestors, ParentOfStrict returns zero values
// and false.
func (m *PrefixMap[T]) ParentOfStrict(p netip.Prefix) (netip.Prefix, T, bool) {
	return m.parentOf(p, true)
}

// ToMap returns a map of all Prefixes in m to their associated values.
func (m *PrefixMap[T]) ToMap() map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree.walk(key{}, func(n *tree[T]) bool {
		if n.hasValue {
			res[prefixFromKey(n.key)] = n.value
		}
		return false
	})
	return res
}

// DescendantsOf returns all descendants of the provided Prefix (including the
// Prefix itself, if it has a value) as a map of Prefixes to values.
func (m PrefixMap[T]) DescendantsOf(p netip.Prefix) *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.descendantsOf(keyFromPrefix(p), false)}
}

// DescendantsOfStrict returns all descendants of the provided Prefix as a map
// of Prefixes to values.
func (m PrefixMap[T]) DescendantsOfStrict(p netip.Prefix) *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.descendantsOf(keyFromPrefix(p), true)}
}

// AncestorsOf returns all ancestors of the provided Prefix (including the
// Prefix itself, if it has a value) as a map of Prefixes to values.
func (m *PrefixMap[T]) AncestorsOf(p netip.Prefix) *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.ancestorsOf(keyFromPrefix(p), false)}
}

// AncestorsOfStrict returns all ancestors of the provided Prefix as a map of
// Prefixes to values.
func (m *PrefixMap[T]) AncestorsOfStrict(p netip.Prefix) *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.ancestorsOf(keyFromPrefix(p), true)}
}
