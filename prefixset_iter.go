//go:build go1.23

package netipds

import (
	"iter"
	"net/netip"
)

// concat returns an iterator which yields from all seqs, in order.
func concat[E any](seqs ...iter.Seq[E]) iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// All returns an iterator over all Prefixes in s.
func (s *PrefixSet) All() iter.Seq[netip.Prefix] {
	return concat(
		s.All4(),
		s.All6(),
	)
}

// All4 returns an iterator over all IPv4 Prefixes in s.
func (s *PrefixSet) All4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		i := 0
		s.tree4.walk(key[keyBits4]{}, func(n *tree[bool, keyBits4]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.ToPrefix())
				i++
			}
			return !canYield || i >= s.size4
		})
		i = 0
	}
}

// All6 returns an iterator over all IPv6 Prefixes in s.
func (s *PrefixSet) All6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		i := 0
		s.tree6.walk(key[keyBits6]{}, func(n *tree[bool, keyBits6]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.ToPrefix())
				i++
			}
			return !canYield || i >= s.size6
		})
		i = 0
	}
}

// AllCompact4 returns an iterator over the prefixes in s
// that are not children of any other prefix in s.
//
// Note: AllCompact4 does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. 1.2.3.0/32 and 1.2.3.1/32.
func (s *PrefixSet) AllCompact4() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		s.tree4.walk(key[keyBits4]{}, func(n *tree[bool, keyBits4]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.ToPrefix())
				return true
			}
			return !canYield
		})
	}
}

// AllCompact6 returns an iterator over the prefixes in s
// that are not children of any other prefix in s.
//
// Note: AllCompact6 does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. ::0/128 and ::1/128.
func (s *PrefixSet) AllCompact6() iter.Seq[netip.Prefix] {
	return func(yield func(netip.Prefix) bool) {
		canYield := true
		s.tree6.walk(key[keyBits6]{}, func(n *tree[bool, keyBits6]) bool {
			if canYield && n.hasEntry {
				canYield = yield(n.key.ToPrefix())
				return true
			}
			return !canYield
		})
	}
}

// AllCompact returns an iterator over the prefixes in s
// that are not children of any other prefix in s.
//
// Note: AllCompact does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. ::0/128 and ::1/128.
func (s *PrefixSet) AllCompact() iter.Seq[netip.Prefix] {
	return concat(
		s.AllCompact4(),
		s.AllCompact6(),
	)
}
