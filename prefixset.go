package netipds

import (
	"fmt"
	"net/netip"
)

type PrefixSetBuilder struct {
	Lazy bool
	tree tree[bool]
}

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

func (s *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.remove(keyFromPrefix(p))
	return nil
}

// Filter removes all Prefixes from s that are not encompassed by pm.
func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree.filter(o.tree)
}

// Subtract modifies the map such that the provided Prefix and all of its
// descendants are removed from the set, leaving behind any remaining parts
// of affected elements. This may add elements to the set to fill in gaps
// around the subtracted Prefix.
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
	t := s.tree.copy()
	if s.Lazy && t != nil {
		t = t.compress()
	}
	return &PrefixSet{*t, t.size()}
}

func (s *PrefixSetBuilder) String() string {
	return s.tree.stringHelper("", "", true)
}

type PrefixSet struct {
	tree tree[bool]
	size int
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

func (s *PrefixSet) Size() int {
	return s.size
}
