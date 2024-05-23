package netipmap

import (
	"fmt"
	"net/netip"
)

// PrefixMapBuilder builds an immutable PrefixMap.
//
// The zero value is a valid PrefixMapBuilder representing a builder with zero
// Prefixes.
//
// Call PrefixMap to obtain an immutable PrefixMap from a PrefixMapBuilder.
type PrefixMapBuilder[T any] struct {
	tree tree[T]
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMapBuilder[T]) Get(p netip.Prefix) (T, bool) {
	return m.tree.get(keyFromPrefix(p))
}

// Set associates the provided value with the provided Prefix.
func (m *PrefixMapBuilder[T]) Set(p netip.Prefix, value T) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	// TODO so should m.tree just be a *tree[T]?
	m.tree = *(m.tree.insert(keyFromPrefix(p), value))
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

// Subtract modifies the map such that the provided Prefix and all of its
// descendants are removed from the map, leaving behind any remaining portions
// of affected Prefixes. This may add entries to the map to fill in gaps around
// the subtracted Prefix.
//
// For example, if m is {::0/126:true}, and we subtract ::0/128, then m will
// become {::1/128:true, ::2/127:true}.
func (m *PrefixMapBuilder[T]) Subtract(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.subtract(keyFromPrefix(p))
	return nil
}

// Filter removes all Prefixes from m that are not encompassed by the provided
// PrefixSet.
func (m *PrefixMapBuilder[T]) Filter(s *PrefixSet) {
	m.tree.filter(s.tree)
}

// PrefixMap returns an immutable PrefixMap representing the current state of m.
//
// The builder remains usable after calling PrefixMap.
func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.copy()}
}

func (s *PrefixMapBuilder[T]) String() string {
	return s.tree.stringHelper("", "", false)
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

// prefixFromKey returns the Prefix represented by the provided key.
func prefixFromKey(b key) netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], b.content.hi)
	bePutUint64(a16[8:], b.content.lo)
	addr := netip.AddrFrom16(a16)
	bits := int(b.len)
	if addr.Is4In6() {
		bits -= 96
	}
	return netip.PrefixFrom(addr.Unmap(), bits)
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
func (m *PrefixMap[T]) DescendantsOf(p netip.Prefix) *PrefixMap[T] {
	return &PrefixMap[T]{*m.tree.descendantsOf(keyFromPrefix(p), false)}
}

// DescendantsOfStrict returns all descendants of the provided Prefix as a map
// of Prefixes to values.
func (m *PrefixMap[T]) DescendantsOfStrict(p netip.Prefix) *PrefixMap[T] {
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

func (m *PrefixMap[T]) String() string {
	return m.tree.stringHelper("", "", false)
}
