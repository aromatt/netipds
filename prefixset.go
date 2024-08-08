package netipds

import (
	"fmt"
	"net/netip"
)

// PrefixSetBuilder builds an immutable PrefixSet.
//
// The zero value is a valid PrefixSetBuilder representing a builder with zero
// Prefixes.
//
// Call PrefixSet to obtain an immutable PrefixSet from a PrefixSetBuilder.
//
// If Lazy == true, then path compression is delayed until a PrefixSet is
// created. The builder itself remains uncompressed. Lazy mode can dramatically
// improve performance when building large PrefixSets.
type PrefixSetBuilder struct {
	Lazy bool
	tree tree[bool]
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	if s.Lazy {
		s.tree = *(s.tree.insertLazy(keyFromPrefix(p), true))
	} else {
		s.tree = *(s.tree.insert(keyFromPrefix(p), true))
	}
	return nil
}

// Remove removes p from s. Only the exact Prefix provided is removed;
// descendants are not.
//
// See RemoveDescendants and Subtract for descendant-removal.
func (s *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.remove(keyFromPrefix(p))
	return nil
}

// Filter removes all Prefixes that are not encompassed by o.
func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree.filter(&o.tree)
}

// Subtract modifies s so that p and all of its descendants are removed,
// leaving behind any remaining portions of affected Prefixes. This may add
// elements to fill in gaps around the subtracted Prefix.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) Subtract(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.subtractKey(keyFromPrefix(p))
	return nil
}

// SubtractSet modifies s so that the Prefixes in o, and all of their
// descendants, are removed from s, leaving behind any remaining portions of
// affected Prefixes. This may add elements to fill in gaps around the
// subtracted Prefixes.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) SubtractSet(o *PrefixSet) {
	s.tree = *s.tree.subtractTree(&o.tree)
}

// IntersectSet modifies s so that it contains the intersection of the entries
// in s and o: each Prefix must either (a) exist in both sets or (b) exist in
// one set and have an ancestor in the other.
func (s *PrefixSetBuilder) IntersectSet(o *PrefixSet) {
	s.tree = *s.tree.intersectTree(&o.tree)
}

// UnionSet modifies s so that it contains the union of the entries in s and o.
func (s *PrefixSetBuilder) UnionSet(o *PrefixSet) {
	s.tree = *s.tree.unionTree(&o.tree)
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	t := s.tree.copy()
	if s.Lazy && t != nil {
		t = t.compress()
	}
	return &PrefixSet{*t, t.size()}
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSetBuilder) String() string {
	return s.tree.stringImpl("", "", true)
}

// PrefixSet is a set of netip.Prefixes. It is implemented as a binary radix
// tree with path compression.
//
// Use PrefixSetBuilder to construct PrefixSets.
type PrefixSet struct {
	tree tree[bool]
	size int
}

// Contains returns true if this set includes the exact Prefix provided.
func (s *PrefixSet) Contains(p netip.Prefix) bool {
	return s.tree.contains(keyFromPrefix(p))
}

// Encompasses returns true if this set includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), false)
}

// EncompassesStrict returns true if this set includes a Prefix which
// completely encompasses p. The encompassing Prefix must be an ancestor of p,
// not p itself.
func (s *PrefixSet) EncompassesStrict(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), true)
}

// Overlaps returns true if this set includes a Prefix which overlaps p.
func (s *PrefixSet) Overlaps(p netip.Prefix) bool {
	return s.tree.overlapsKey(keyFromPrefix(p))
}

// Prefixes returns a slice of all Prefixes in s.
func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, s.tree.size())
	i := 0
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasEntry {
			res[i] = prefixFromKey(n.key)
			i++
		}
		return i >= len(res)
	})
	return res
}

// PrefixesCompact returns a slice of the Prefixes in s that are not
// children of other Prefixes in s.
//
// Note: PrefixCompact does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. 1.2.3.0/32 and 1.2.3.1/32.
func (s *PrefixSet) PrefixesCompact() []netip.Prefix {
	res := make([]netip.Prefix, 0, s.tree.size())
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasEntry {
			res = append(res, prefixFromKey(n.key))
			return true
		}
		return false
	})
	return res
}

// String returns a human-readable representation of the s's tree structure.
func (s *PrefixSet) String() string {
	return s.tree.stringImpl("", "", true)
}

// Size returns the number of elements in s.
func (s *PrefixSet) Size() int {
	return s.size
}
