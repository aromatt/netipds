package netipds

import (
	"fmt"
	"net/netip"
)

// PrefixSetBuilder builds an immutable [PrefixSet].
//
// TODO: we have lost this property
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

func NewPrefixSetBuilder() *PrefixSetBuilder {
	return &PrefixSetBuilder{
		tree: *newTree[bool](1, 1),
	}
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	if s.Lazy {
		s.tree.Cursor().InsertLazy(keyFromPrefix(p), true)
	} else {
		s.tree.Cursor().Insert(keyFromPrefix(p), true)
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
	s.tree.Cursor().Remove(keyFromPrefix(p))
	return nil
}

// Filter removes all Prefixes that are not encompassed by o from s.
func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree.Cursor().Filter(o.tree.Cursor())
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
	s.tree.Cursor().SubtractKey(keyFromPrefix(p))
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
	s.tree.Cursor().SubtractTree(o.tree.Cursor())
}

// Intersect modifies s so that it contains the intersection of the entries
// in s and o: to be included in the result, a Prefix must either (a) exist in
// both sets or (b) exist in one set and have an ancestor in the other.
func (s *PrefixSetBuilder) Intersect(o *PrefixSet) {
	s.tree.Cursor().IntersectTree(o.tree.Cursor())
}

// Merge modifies s so that it contains the union of the entries in s and o.
func (s *PrefixSetBuilder) Merge(o *PrefixSet) {
	s.tree.Cursor().MergeTree(o.tree.Cursor())
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	t := s.tree.Copy()
	if s.Lazy {
		t.Cursor().Compress()
	}
	return &PrefixSet{*t, t.Cursor().Size()}
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSetBuilder) String() string {
	return s.tree.Cursor().stringImpl("", "", true)
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
	tree tree[bool]
	size int
}

// Contains returns true if this set includes the exact Prefix provided.
func (s *PrefixSet) Contains(p netip.Prefix) bool {
	return s.tree.Cursor().Contains(keyFromPrefix(p))
}

// Encompasses returns true if this set includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	return s.tree.Cursor().Encompasses(keyFromPrefix(p), false)
}

// EncompassesStrict returns true if this set includes a Prefix which
// completely encompasses p. The encompassing Prefix must be an ancestor of p,
// not p itself.
func (s *PrefixSet) EncompassesStrict(p netip.Prefix) bool {
	return s.tree.Cursor().Encompasses(keyFromPrefix(p), true)
}

// OverlapsPrefix returns true if this set includes a Prefix which overlaps p.
func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	return s.tree.Cursor().OverlapsKey(keyFromPrefix(p))
}

func (s *PrefixSet) rootOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, ok bool) {
	label, _, ok := s.tree.Cursor().RootOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, false
	}
	return label.toPrefix(), true
}

// RootOf returns the shortest-prefix ancestor of p in s, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
func (s *PrefixSet) RootOf(p netip.Prefix) (netip.Prefix, bool) {
	return s.rootOf(p, false)
}

// RootOfStrict returns the shortest-prefix ancestor of p in s, if any. If p
// has no ancestors in s, then RootOfStrict returns zero values and false.
func (s *PrefixSet) RootOfStrict(p netip.Prefix) (netip.Prefix, bool) {
	return s.rootOf(p, true)
}

func (s *PrefixSet) parentOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, ok bool) {
	key, _, ok := s.tree.Cursor().ParentOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, false
	}
	return key.toPrefix(), true
}

// ParentOf returns the longest-prefix ancestor of p in s, if any. If p itself
// has an entry, then p's entry is returned.
func (s *PrefixSet) ParentOf(p netip.Prefix) (netip.Prefix, bool) {
	return s.parentOf(p, false)
}

// ParentOfStrict returns the longest-prefix ancestor of p in s, if any.
// If p has no ancestors in the set, then ParentOfStrict returns zero values
// and false.
func (s *PrefixSet) ParentOfStrict(p netip.Prefix) (netip.Prefix, bool) {
	return s.parentOf(p, true)
}

// DescendantsOf returns a PrefixSet containing all descendants of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) DescendantsOf(p netip.Prefix) *PrefixSet {
	t := s.tree.Cursor().DescendantsOf(keyFromPrefix(p), false)
	return &PrefixSet{*t.tree, t.Size()}
}

// DescendantsOfStrict returns a PrefixSet containing all descendants of p in
// s, excluding p itself.
func (s *PrefixSet) DescendantsOfStrict(p netip.Prefix) *PrefixSet {
	t := s.tree.Cursor().DescendantsOf(keyFromPrefix(p), true)
	return &PrefixSet{*t.tree, t.Size()}
}

// AncestorsOf returns a PrefixSet containing all ancestors of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) AncestorsOf(p netip.Prefix) *PrefixSet {
	t := s.tree.Cursor().AncestorsOf(keyFromPrefix(p), false)
	return &PrefixSet{*t.tree, t.Size()}
}

// AncestorsOfStrict returns a PrefixSet containing all ancestors of p in s,
// excluding p itself.
func (s *PrefixSet) AncestorsOfStrict(p netip.Prefix) *PrefixSet {
	t := s.tree.Cursor().AncestorsOf(keyFromPrefix(p), true)
	return &PrefixSet{*t.tree, t.Size()}
}

// Prefixes returns a slice of all Prefixes in s.
func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, s.size)
	i := 0
	s.tree.Cursor().walk(key{}, func(n treeCursor[bool]) bool {
		if n.HasEntry() {
			res[i] = n.Key().toPrefix()
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
	res := make([]netip.Prefix, 0, s.size)
	s.tree.Cursor().walk(key{}, func(n treeCursor[bool]) bool {
		if n.HasEntry() {
			res = append(res, n.Key().toPrefix())
			return true
		}
		return false
	})
	return res
}

// String returns a human-readable representation of the s's tree structure.
func (s *PrefixSet) String() string {
	return s.tree.Cursor().stringImpl("", "", true)
}

// Size returns the number of elements in s.
func (s *PrefixSet) Size() int {
	return s.size
}
