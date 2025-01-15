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
		s.tree.Cursor().walk(key{}, func(n treeCursor[bool]) bool {
			if canYield && n.HasEntry() {
				canYield = yield(n.Key().toPrefix())
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
		s.tree.Cursor().walk(key{}, func(n treeCursor[bool]) bool {
			if canYield && n.HasEntry() {
				canYield = yield(n.Key().toPrefix())
				return true
			}
			return !canYield
		})
	}
}
