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
type PrefixSetBuilder struct {
	tree tree[bool]
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree = *s.tree.insert(keyFromPrefix(p), true)
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
	s.tree.filter(o.tree)
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
	s.tree.subtract(keyFromPrefix(p))
	return nil
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	return &PrefixSet{*s.tree.copy()}
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSetBuilder) String() string {
	return s.tree.stringHelper("", "", true)
}

// PrefixSet is a set of netip.Prefixes. It is implemented as a binary radix
// tree with path compression.
//
// Use PrefixSetBuilder to construct PrefixSets.
type PrefixSet struct {
	tree tree[bool]
}

func (s *PrefixSet) Contains(p netip.Prefix) bool {
	return s.tree.contains(keyFromPrefix(p))
}

func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), false)
}

func (s *PrefixSet) EncompassesStrict(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), true)
}

func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, s.tree.size())
	i := 0
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasValue {
			res[i] = prefixFromKey(n.key)
			i++
		}
		return i >= len(res)
	})
	return res
}

func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	return s.tree.overlapsKey(keyFromPrefix(p))
}

// SubtractFromPrefix returns a new PrefixSet that is the result of removing
// all Prefixes in s that are encompassed by p, including p itself.
func (s *PrefixSet) SubtractFromPrefix(p netip.Prefix) *PrefixSet {
	ret := &PrefixSetBuilder{}
	ret.Add(p)
	s.tree.walk(keyFromPrefix(p), func(n *tree[bool]) bool {
		ret.Subtract(prefixFromKey(n.key))
		return false
	})
	return ret.PrefixSet()
}

// PrettyPrint prints the PrefixSet in a human-readable format.
func (s *PrefixSet) String() string {
	return s.tree.stringHelper("", "", true)
}
