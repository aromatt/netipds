package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixSetBuilder struct {
	tree tree[bool]
}

func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree = *s.tree.insert(keyFromPrefix(p), true)
	return nil
}

func (s *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.remove(keyFromPrefix(p))
	return nil
}

func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree.filter(o.tree)
}

func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	return &PrefixSet{*s.tree.copy()}
}

// RemoveDescendants removes all Prefixes from s that are encompassed by the
// provided Prefix, including the Prefix itself.
func (s *PrefixSetBuilder) RemoveDescendants(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.removeDescendants(keyFromPrefix(p), false)
	return nil
}

// RemoveDescendantsStrict removes all Prefixes from m that are encompassed by
// the provided Prefix. The provided Prefix itself is not removed.
func (s *PrefixSetBuilder) RemoveDescendantsStrict(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.removeDescendants(keyFromPrefix(p), true)
	return nil
}

func (s *PrefixSetBuilder) String() string {
	return s.tree.stringHelper("", "", true)
}

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
	res := make([]netip.Prefix, 0)
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasValue {
			res = append(res, prefixFromKey(n.key))
		}
		return false
	})
	return res
}

func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	return s.tree.overlapsKey(keyFromPrefix(p))
}

// SubtractFromPrefix returns a new PrefixSet that is the result of removing all
// Prefixes in s that are encompassed by the provided Prefix. The provided
// Prefix itself is not removed.
func (s *PrefixSet) SubtractFromPrefix(p netip.Prefix) *PrefixSet {
	ret := &PrefixSetBuilder{}
	ret.Add(p)
	pk := keyFromPrefix(p)
	s.tree.walk(pk, func(n *tree[bool]) bool {
		ret.RemoveDescendants(prefixFromKey(n.key))
		return false
	})
	return ret.PrefixSet()
}

// PrettyPrint prints the PrefixSet in a human-readable format.
func (s *PrefixSet) String() string {
	return s.tree.stringHelper("", "", true)
}
