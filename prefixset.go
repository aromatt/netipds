package netipmap

import (
	"fmt"
	"net/netip"
)

type PrefixSetBuilder struct {
	tree tree[bool]
}

func (m *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.set(keyFromPrefix(p), true)
	return nil
}

func (m *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	m.tree.remove(keyFromPrefix(p))
	return nil
}

func (m *PrefixSetBuilder) Filter(s *PrefixSet) {
	m.tree.filter(s.tree)
}

func (m *PrefixSetBuilder) PrefixSet() *PrefixSet {
	return &PrefixSet{*m.tree.copy()}
}

type PrefixSet struct {
	tree tree[bool]
}

func (m *PrefixSet) Contains(p netip.Prefix) bool {
	return m.tree.contains(keyFromPrefix(p))
}

func (m *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, 0)
	m.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasValue {
			res = append(res, prefixFromKey(n.key))
		}
		return false
	})
	return res
}
