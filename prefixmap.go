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
	tree6 tree[T, keyBits6]
	tree4 tree[T, keyBits4]
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMapBuilder[T]) Get(p netip.Prefix) (T, bool) {
	if p.Addr().Is4() {
		return m.tree4.get(key4FromPrefix(p.Masked()))
	} else {
		return m.tree6.get(key6FromPrefix(p.Masked()))
	}
}

// Set associates v with p.
func (m *PrefixMapBuilder[T]) Set(p netip.Prefix, v T) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
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
		return fmt.Errorf("Prefix is not valid: %v", p)
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
	m.tree6.filter(&s.tree6)
	m.tree4.filter(&s.tree4)
}

// PrefixMap returns an immutable PrefixMap representing the current state of m.
//
// The builder remains usable after calling PrefixMap.
func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	t6 := m.tree6.copy()
	t4 := m.tree4.copy()
	return &PrefixMap[T]{*t6, *t4, t6.size(), t4.size()}
}

func (s *PrefixMapBuilder[T]) String() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", false),
		s.tree6.stringImpl("", "", false),
	)
}

// PrefixMap is a map of [netip.Prefix] to T. It is implemented as a binary
// radix tree.
//
// Use [PrefixMapBuilder] to construct PrefixMaps.
type PrefixMap[T any] struct {
	tree6 tree[T, keyBits6]
	tree4 tree[T, keyBits4]
	size6 int
	size4 int
}

// Get returns the value associated with the exact Prefix provided, if any.
func (m *PrefixMap[T]) Get(p netip.Prefix) (T, bool) {
	if p.Addr().Is4() {
		return m.tree4.get(key4FromPrefix(p))
	} else {
		return m.tree6.get(key6FromPrefix(p))
	}
}

// Contains returns true if this map includes the exact Prefix provided.
func (m *PrefixMap[T]) Contains(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.contains(key4FromPrefix(p))
	} else {
		return m.tree6.contains(key6FromPrefix(p))
	}
}

// Encompasses returns true if this map includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (m *PrefixMap[T]) Encompasses(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.encompasses(key4FromPrefix(p))
	} else {
		return m.tree6.encompasses(key6FromPrefix(p))
	}
}

// OverlapsPrefix returns true if this map includes a Prefix which overlaps p.
func (m *PrefixMap[T]) OverlapsPrefix(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return m.tree4.overlapsKey(key4FromPrefix(p))
	} else {
		return m.tree6.overlapsKey(key6FromPrefix(p))
	}
}

// RootOf returns the shortest-prefix ancestor of p in m, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
// TODO repetitive
func (m *PrefixMap[T]) RootOf(p netip.Prefix) (root netip.Prefix, val T, ok bool) {
	if p.Addr().Is4() {
		var k key[keyBits4]
		k, val, ok = m.tree4.rootOf(key4FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	} else {
		var k key[keyBits6]
		k, val, ok = m.tree6.rootOf(key6FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	}
	return
}

// ParentOf returns the longest-prefix ancestor of p in m, if any. If p itself
// has an entry, then p's entry is returned.
// TODO repetitive
func (m *PrefixMap[T]) ParentOf(p netip.Prefix) (parent netip.Prefix, val T, ok bool) {
	if p.Addr().Is4() {
		var k key[keyBits4]
		k, val, ok = m.tree4.parentOf(key4FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}
	} else {
		var k key[keyBits6]
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
	m.tree4.walk(key[keyBits4]{}, func(n *tree[T, keyBits4]) bool {
		if n.hasEntry {
			res[n.key.ToPrefix()] = n.value
		}
		return false
	})
	m.tree6.walk(key[keyBits6]{}, func(n *tree[T, keyBits6]) bool {
		if n.hasEntry {
			res[n.key.ToPrefix()] = n.value
		}
		return false
	})

	return res
}

// DescendantsOf returns a PrefixMap containing all descendants of p in m,
// including p itself if it has an entry.
// TODO repetition
func (m *PrefixMap[T]) DescendantsOf(p netip.Prefix) *PrefixMap[T] {
	if p.Addr().Is4() {
		t := m.tree4.descendantsOf(key4FromPrefix(p), false)
		return &PrefixMap[T]{tree4: *t, size4: t.size()}
	} else {
		t := m.tree6.descendantsOf(key6FromPrefix(p), false)
		return &PrefixMap[T]{tree6: *t, size6: t.size()}
	}
}

// AncestorsOf returns a PrefixMap containing all ancestors of p in m,
// including p itself if it has an entry.
// TODO repetition
func (m *PrefixMap[T]) AncestorsOf(p netip.Prefix) *PrefixMap[T] {
	if p.Addr().Is4() {
		t := m.tree4.ancestorsOf(key4FromPrefix(p), false)
		return &PrefixMap[T]{tree4: *t, size4: t.size()}
	} else {
		t := m.tree6.ancestorsOf(key6FromPrefix(p), false)
		return &PrefixMap[T]{tree6: *t, size6: t.size()}
	}

}

// Filter returns a new PrefixMap containing the entries of m that are
// encompassed by s.
func (m *PrefixMap[T]) Filter(s *PrefixSet) *PrefixMap[T] {
	t6 := m.tree6.filterCopy(&s.tree6)
	t4 := m.tree4.filterCopy(&s.tree4)
	return &PrefixMap[T]{*t6, *t4, t6.size(), t4.size()}
}

// String returns a human-readable representation of m's tree structure.
func (s *PrefixMap[T]) String() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", false),
		s.tree6.stringImpl("", "", false),
	)
}

// Size returns the number of entries in m.
func (m *PrefixMap[T]) Size() int {
	return m.size6 + m.size4
}
