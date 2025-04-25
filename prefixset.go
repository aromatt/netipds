package netipds

import (
	"fmt"
	"net/netip"
)

// PrefixSetBuilder builds an immutable [PrefixSet].
//
// The zero value is a valid PrefixSetBuilder representing a builder with zero
// Prefixes.
//
// Call PrefixSet to obtain an immutable PrefixSet from a PrefixSetBuilder.
type PrefixSetBuilder struct {
	tree6 tree[bool, keyBits6]
	tree4 tree[bool, keyBits4]
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
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
		return fmt.Errorf("Prefix is not valid: %v", p)
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
	s.tree6.filter(&o.tree6)
	s.tree4.filter(&o.tree4)
}

// SubtractPrefix modifies s so that p and all of its descendants are removed,
// leaving behind any remaining portions of affected Prefixes. This may add
// elements to fill in gaps around the subtracted Prefix.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) SubtractPrefix(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
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
	s.tree6 = *s.tree6.intersectTree(&o.tree6)
	s.tree4 = *s.tree4.intersectTree(&o.tree4)
}

// Merge modifies s so that it contains the union of the entries in s and o.
func (s *PrefixSetBuilder) Merge(o *PrefixSet) {
	s.tree6 = *s.tree6.mergeTree(&o.tree6)
	s.tree4 = *s.tree4.mergeTree(&o.tree4)
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	t6 := s.tree6.copy()
	t4 := s.tree4.copy()
	return &PrefixSet{*t6, *t4, t6.size(), t4.size()}
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSetBuilder) String() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", true),
		s.tree6.stringImpl("", "", true),
	)
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
	tree6 tree[bool, keyBits6]
	tree4 tree[bool, keyBits4]
	size6 int
	size4 int
}

// Contains returns true if this set includes the exact Prefix provided.
func (s *PrefixSet) Contains(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.contains(key4FromPrefix(p))
	} else {
		return s.tree6.contains(key6FromPrefix(p))
	}

}

// Encompasses returns true if this set includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.encompasses(key4FromPrefix(p))
	} else {
		return s.tree6.encompasses(key6FromPrefix(p))
	}
}

// OverlapsPrefix returns true if this set includes a Prefix which overlaps p.
func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return s.tree4.overlapsKey(key4FromPrefix(p))
	} else {
		return s.tree6.overlapsKey(key6FromPrefix(p))
	}
}

// RootOf returns the shortest-prefix ancestor of p in s, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
func (s *PrefixSet) RootOf(p netip.Prefix) (root netip.Prefix, ok bool) {
	if p.Addr().Is4() {
		var k key[keyBits4]
		k, _, ok = s.tree4.rootOf(key4FromPrefix(p))
		if ok {
			root = k.ToPrefix()
		}
	} else {
		var k key[keyBits6]
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
		var k key[keyBits4]
		k, _, ok = s.tree4.parentOf(key4FromPrefix(p))
		if ok {
			parent = k.ToPrefix()
		}
	} else {
		var k key[keyBits6]
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
		t := s.tree4.descendantsOf(key4FromPrefix(p), false)
		return &PrefixSet{tree4: *t, size4: t.size()}
	} else {
		t := s.tree6.descendantsOf(key6FromPrefix(p), false)
		return &PrefixSet{tree6: *t, size6: t.size()}
	}
}

// AncestorsOf returns a PrefixSet containing all ancestors of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) AncestorsOf(p netip.Prefix) *PrefixSet {
	if p.Addr().Is4() {
		t := s.tree4.ancestorsOf(key4FromPrefix(p), false)
		return &PrefixSet{tree4: *t, size4: t.size()}
	} else {
		t := s.tree6.ancestorsOf(key6FromPrefix(p), false)
		return &PrefixSet{tree6: *t, size6: t.size()}
	}
}

// Prefixes returns a slice of all Prefixes in s.
func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, 0, s.size6+s.size4)
	s.tree6.walk(key[keyBits6]{}, func(n *tree[bool, keyBits6]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
		}
		return len(res) == s.size6
	})
	s.tree4.walk(key[keyBits4]{}, func(n *tree[bool, keyBits4]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
		}
		return len(res) == cap(res)
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
	s.tree6.walk(key[keyBits6]{}, func(n *tree[bool, keyBits6]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			return true
		}
		return false
	})
	s.tree4.walk(key[keyBits4]{}, func(n *tree[bool, keyBits4]) bool {
		if n.hasEntry {
			res = append(res, n.key.ToPrefix())
			return true
		}
		return false
	})
	return res
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSet) String() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", true),
		s.tree6.stringImpl("", "", true),
	)
}

// Size returns the number of elements in s.
func (s *PrefixSet) Size() int {
	return s.size6 + s.size4
}
