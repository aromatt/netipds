// Package netipds builds on the net/netip & go4.org/netipx family of packages
// by adding two immutable, trie-based collection types for netip.Prefix:
//
//   - PrefixMap[T] - for associating data with IPs and prefixes and fetching
//     that data with network hierarchy awareness
//   - PrefixSet - for storing sets of prefixes and combining those sets in
//     useful ways (unions, intersections, etc)
//
// Both are backed by a binary radix tree, which enables a rich set of efficient
// queries about prefix containment, hierarchy, and overlap.
package netipds

import (
	"fmt"
	"net/netip"
)

// PrefixMapBuilder builds an immutable [PrefixMap].
//
// The zero value is a valid PrefixMapBuilder representing a builder with zero
// Prefixes.
//
// Call [PrefixMapBuilder.PrefixMap] to obtain an immutable PrefixMap from a
// PrefixMapBuilder.
type PrefixMapBuilder[T any] struct {
	tree4 tree[T, keybits4]
	tree6 tree[T, keybits6]
}

// Set associates v with p.
func (m *PrefixMapBuilder[T]) Set(p netip.Prefix, v T) error {
	if !p.IsValid() {
		return fmt.Errorf("prefix is not valid: %v", p)
	}
	if p.Addr().Is4() {
		m.tree4 = *(m.tree4.insert(key4FromPrefix(p.Masked()), v))
	} else {
		m.tree6 = *(m.tree6.insert(key6FromPrefix(p.Masked()), v))
	}
	return nil
}

// Remove removes p from m. Only the exact Prefix provided is removed;
// descendants are not.
//
// To remove entire sections of IP space at once, see
// [PrefixMapBuilder.Filter].
func (m *PrefixMapBuilder[T]) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("prefix is not valid: %v", p)
	}
	if p.Addr().Is4() {
		m.tree4.remove(key4FromPrefix(p.Masked()))
	} else {
		m.tree6.remove(key6FromPrefix(p.Masked()))
	}
	return nil
}

// Filter removes all Prefixes that are not encompassed by s from m.
func (m *PrefixMapBuilder[T]) Filter(s *PrefixSet) {
	m.tree4.filter(&s.tree4)
	m.tree6.filter(&s.tree6)
}

// PrefixMap returns an immutable PrefixMap representing the current state of m.
//
// The builder remains usable after calling PrefixMap.
func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	t4 := m.tree4.copy()
	t6 := m.tree6.copy()
	return &PrefixMap[T]{*t4, *t6, t4.size(), t6.size()}
}

// PrefixMap is a map of [netip.Prefix] to T. It is implemented as a binary
// radix tree.
//
// Use [PrefixMapBuilder] to construct PrefixMaps.
type PrefixMap[T any] struct {
	tree4 tree[T, keybits4]
	tree6 tree[T, keybits6]
	size4 int
	size6 int
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMap[T]) Get(p netip.Prefix) (T, bool) {
	if p.Addr().Is4() {
		return m.tree4.get(key4FromPrefix(p))
	}
	return m.tree6.get(key6FromPrefix(p))
}

// Contains returns true if this map includes the exact Prefix provided.
func (m *PrefixMap[T]) Contains(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.contains(key4FromPrefix(p))
	}
	return m.tree6.contains(key6FromPrefix(p))
}

// Encompasses returns true if this map includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (m *PrefixMap[T]) Encompasses(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.encompasses(key4FromPrefix(p))
	}
	return m.tree6.encompasses(key6FromPrefix(p))
}

// OverlapsPrefix returns true if this map includes a Prefix which overlaps p.
func (m *PrefixMap[T]) OverlapsPrefix(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.overlapsKey(key4FromPrefix(p))
	}
	return m.tree6.overlapsKey(key6FromPrefix(p))
}

// RootOf returns the shortest-prefix ancestor of p in m, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
func (m *PrefixMap[T]) RootOf(p netip.Prefix) (root netip.Prefix, val T, ok bool) {
	if p.Addr().Is4() {
		var k key[keybits4]
		k, val, ok = m.tree4.rootOf(key4FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	} else {
		var k key[keybits6]
		k, val, ok = m.tree6.rootOf(key6FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	}
	return
}

// ParentOf returns the longest-prefix ancestor of p in m, if any. If p itself
// has an entry, then p's entry is returned.
func (m *PrefixMap[T]) ParentOf(p netip.Prefix) (parent netip.Prefix, val T, ok bool) {
	if p.Addr().Is4() {
		var k key[keybits4]
		k, val, ok = m.tree4.parentOf(key4FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}
	} else {
		var k key[keybits6]
		k, val, ok = m.tree6.parentOf(key6FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}

	}
	return
}

// ToMap returns a map of all Prefixes in m to their associated values.
func (m *PrefixMap[T]) ToMap() map[netip.Prefix]T {
	res := make(map[netip.Prefix]T)
	m.tree4.walk(key[keybits4]{}, func(n *tree[T, keybits4]) bool {
		if n.hasEntry {
			res[n.key.ToPrefix()] = n.value
		}
		return false
	})
	m.tree6.walk(key[keybits6]{}, func(n *tree[T, keybits6]) bool {
		if n.hasEntry {
			res[n.key.ToPrefix()] = n.value
		}
		return false
	})
	return res
}

// DescendantsOf returns a PrefixMap containing all descendants of p in m,
// including p itself if it has an entry.
func (m *PrefixMap[T]) DescendantsOf(p netip.Prefix) *PrefixMap[T] {
	if p.Addr().Is4() {
		t := m.tree4.descendantsOf(key4FromPrefix(p))
		return &PrefixMap[T]{tree4: *t, size4: t.size()}
	}
	t := m.tree6.descendantsOf(key6FromPrefix(p))
	return &PrefixMap[T]{tree6: *t, size6: t.size()}
}

// AncestorsOf returns a PrefixMap containing all ancestors of p in m,
// including p itself if it has an entry.
func (m *PrefixMap[T]) AncestorsOf(p netip.Prefix) *PrefixMap[T] {
	if p.Addr().Is4() {
		t := m.tree4.ancestorsOf(key4FromPrefix(p))
		return &PrefixMap[T]{tree4: *t, size4: t.size()}
	}
	t := m.tree6.ancestorsOf(key6FromPrefix(p))
	return &PrefixMap[T]{tree6: *t, size6: t.size()}

}

// Filter returns a new PrefixMap containing the entries of m that are
// encompassed by s.
func (m *PrefixMap[T]) Filter(s *PrefixSet) *PrefixMap[T] {
	t4 := m.tree4.filterCopy(&s.tree4)
	t6 := m.tree6.filterCopy(&s.tree6)
	return &PrefixMap[T]{*t4, *t6, t4.size(), t6.size()}
}

// Size returns the number of entries in m.
func (m *PrefixMap[T]) Size() int {
	return m.size4 + m.size6
}

// PrefixSetBuilder builds an immutable [PrefixSet].
//
// The zero value is a valid PrefixSetBuilder representing a builder with zero
// Prefixes.
//
// Call PrefixSet to obtain an immutable PrefixSet from a PrefixSetBuilder.
type PrefixSetBuilder struct {
	tree4 tree[bool, keybits4]
	tree6 tree[bool, keybits6]
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("prefix is not valid: %v", p)
	}
	if p.Addr().Is4() {
		s.tree4 = *(s.tree4.insert(key4FromPrefix(p.Masked()), true))
	} else {
		s.tree6 = *(s.tree6.insert(key6FromPrefix(p.Masked()), true))
	}
	return nil
}

// Remove removes p from s. Only the exact Prefix provided is removed;
// descendants are not.
//
// To remove entire sections of IP space at once, see
// [PrefixSetBuilder.Filter], [PrefixSetBuilder.Subtract] and
// [PrefixSetBuilder.SubtractPrefix].
func (s *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("prefix is not valid: %v", p)
	}
	if p.Addr().Is4() {
		s.tree4.remove(key4FromPrefix(p.Masked()))
	} else {
		s.tree6.remove(key6FromPrefix(p.Masked()))
	}
	return nil
}

// Filter removes all Prefixes that are not encompassed by o from s.
//
// When filtering, a Prefix in o has no effect on its parent in s. To remove
// subsets of Prefixes, see [PrefixSetBuilder.Subtract] and
// [PrefixSetBuilder.SubtractPrefix].
func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree4.filter(&o.tree4)
	s.tree6.filter(&o.tree6)
}

// SubtractPrefix modifies s so that p and all of its descendants are removed,
// leaving behind any remaining portions of affected Prefixes. This may add
// elements to fill in gaps around the subtracted Prefix.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) SubtractPrefix(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("prefix is not valid: %v", p)
	}
	if p.Addr().Is4() {
		s.tree4.subtractKey(key4FromPrefix(p.Masked()))
	} else {
		s.tree6.subtractKey(key6FromPrefix(p.Masked()))
	}
	return nil
}

// Subtract modifies s so that the Prefixes in o, and all of their
// descendants, are removed from s, leaving behind any remaining portions of
// affected Prefixes. This may add elements to fill in gaps around the
// subtracted Prefixes.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) Subtract(o *PrefixSet) {
	s.tree4 = *s.tree4.subtractTree(&o.tree4)
	s.tree6 = *s.tree6.subtractTree(&o.tree6)
}

// Intersect modifies s so that it contains the intersection of the entries
// in s and o: to be included in the result, a Prefix must either (a) exist in
// both sets or (b) exist in one set and have an ancestor in the other.
func (s *PrefixSetBuilder) Intersect(o *PrefixSet) {
	s.tree4 = *s.tree4.intersectTree(&o.tree4)
	s.tree6 = *s.tree6.intersectTree(&o.tree6)
}

// Merge modifies s so that it contains the union of the entries in s and o.
func (s *PrefixSetBuilder) Merge(o *PrefixSet) {
	s.tree4 = *s.tree4.mergeTree(&o.tree4)
	s.tree6 = *s.tree6.mergeTree(&o.tree6)
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	t4 := s.tree4.copy()
	t6 := s.tree6.copy()
	return &PrefixSet{*t4, *t6, t4.size(), t6.size()}
}

// PrefixSet is a set of [netip.Prefix] values. It is implemented as a binary
// radix tree.
//
// PrefixSet offers unique functionality beyond what a PrefixMap[bool] can do.
// In particular, during the building stage (PrefixSetBuilder) you can combine
// sets in useful ways using methods like [PrefixSetBuilder.Merge],
// [PrefixSetBuilder.Intersect], and [PrefixSetBuilder.Subtract].
//
// Use [PrefixSetBuilder] to construct PrefixSets.
type PrefixSet struct {
	tree4 tree[bool, keybits4]
	tree6 tree[bool, keybits6]
	size4 int
	size6 int
}

// Contains returns true if this set includes the exact Prefix provided.
func (s *PrefixSet) Contains(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.contains(key4FromPrefix(p))
	}
	return s.tree6.contains(key6FromPrefix(p))
}

// Encompasses returns true if this set includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.encompasses(key4FromPrefix(p))
	}
	return s.tree6.encompasses(key6FromPrefix(p))
}

// OverlapsPrefix returns true if this set includes a Prefix which overlaps p.
func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.overlapsKey(key4FromPrefix(p))
	}
	return s.tree6.overlapsKey(key6FromPrefix(p))
}

// RootOf returns the shortest-prefix ancestor of p in s, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
func (s *PrefixSet) RootOf(p netip.Prefix) (root netip.Prefix, ok bool) {
	if p.Addr().Is4() {
		var k key[keybits4]
		k, _, ok = s.tree4.rootOf(key4FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	} else {
		var k key[keybits6]
		k, _, ok = s.tree6.rootOf(key6FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	}
	return
}

// ParentOf returns the longest-prefix ancestor of p in s, if any. If p itself
// has an entry, then p's entry is returned.
func (s *PrefixSet) ParentOf(p netip.Prefix) (parent netip.Prefix, ok bool) {
	if p.Addr().Is4() {
		var k key[keybits4]
		k, _, ok = s.tree4.parentOf(key4FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}
	} else {
		var k key[keybits6]
		k, _, ok = s.tree6.parentOf(key6FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}
	}
	return
}

// DescendantsOf returns a PrefixSet containing all descendants of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) DescendantsOf(p netip.Prefix) *PrefixSet {
	if p.Addr().Is4() {
		t := s.tree4.descendantsOf(key4FromPrefix(p))
		return &PrefixSet{tree4: *t, size4: t.size()}
	}
	t := s.tree6.descendantsOf(key6FromPrefix(p))
	return &PrefixSet{tree6: *t, size6: t.size()}
}

// AncestorsOf returns a PrefixSet containing all ancestors of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) AncestorsOf(p netip.Prefix) *PrefixSet {
	if p.Addr().Is4() {
		t := s.tree4.ancestorsOf(key4FromPrefix(p))
		return &PrefixSet{tree4: *t, size4: t.size()}
	}
	t := s.tree6.ancestorsOf(key6FromPrefix(p))
	return &PrefixSet{tree6: *t, size6: t.size()}
}

// Prefixes returns a slice of all Prefixes in s.
func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, 0, s.size6+s.size4)
	i := 0
	s.tree4.walk(key[keybits4]{}, func(n *tree[bool, keybits4]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			i++
		}
		return i >= s.size4
	})
	i = 0
	s.tree6.walk(key[keybits6]{}, func(n *tree[bool, keybits6]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			i++
		}
		return i >= s.size6
	})
	return res
}

// PrefixesCompact returns a slice of the Prefixes in s that are not
// children of other Prefixes in s.
//
// Note: PrefixCompact does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. 1.2.3.0/32 and 1.2.3.1/32.
func (s *PrefixSet) PrefixesCompact() []netip.Prefix {
	res := make([]netip.Prefix, 0, s.size6+s.size4)
	s.tree4.walk(key[keybits4]{}, func(n *tree[bool, keybits4]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			return true
		}
		return false
	})
	s.tree6.walk(key[keybits6]{}, func(n *tree[bool, keybits6]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			return true
		}
		return false
	})
	return res
}

// Size returns the number of elements in s.
func (s *PrefixSet) Size() int {
	return s.size4 + s.size6
}
