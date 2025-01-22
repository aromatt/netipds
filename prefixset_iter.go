//go:build go1.23

package netipds

import (
	"iter"
	"net/netip"
)

// All returns an iterator over all prefixes in s.
func (s *PrefixSet) All() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		i := 0
		s.tree.walk(key6{}, func(n *tree[bool]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.toPrefix())
				i++
			}
			return !canYield || i >= s.size
		})
	}
}

// AllCompact returns an iterator over the prefixes in s
// that are not children of any other prefix in s.
//
// Note: AllCompact does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. 1.2.3.0/32 and 1.2.3.1/32.
func (s *PrefixSet) AllCompact() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		s.tree.walk(key6{}, func(n *tree[bool]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.toPrefix())
				return true
			}
			return !canYield
		})
	}
}
