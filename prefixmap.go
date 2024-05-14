package netipmap

import (
	"fmt"
	"net/netip"
)

// label represents one node's key fragment.
type label struct {
	value uint128
	len   uint8
}

func (l label) truncated(n uint8) label {
	return label{l.value.bitsClearedFrom(n), n}
}

func (l label) rest(from uint8) label {
	return label{l.value.shiftLeft(128 - from), l.len - from}
}

type node[T any] struct {
	label label
	value *T
	left  *node[T]
	right *node[T]
}

func NewNode[T any](label label, value *T) *node[T] {
	return &node[T]{label: label, value: value}
}

// getBit returns true if bit at position i is set in u, counting from the
// most-significant bit toward the least.
func getBit(u uint128, i uint8) bool {
	if i < 64 {
		return u.hi&(uint64(1)<<(63-i)) > 0
	}
	return u.lo&(uint64(1)<<(127-i)) > 0
}

// n:         0000
// -              00
// -              11
// new label: 0010
// common:    2
// heir:      left
func (n *node[T]) set(label label, value *T) {
	if label == n.label {
		n.value = value
		return
	}
	common := n.label.value.commonPrefixLen(label.value)

	if common < n.label.len {
		var heir, sibling **node[T]

		// The bit after the common prefix determines which child will handle
		// which suffix. If n.label has 0, then left will inherit n.label's
		// suffix, and right will handle the new suffix.
		if getBit(n.label.value, common) {
			heir = &n.right
			sibling = &n.left
		} else {
			heir = &n.left
			sibling = &n.right
		}

		// heir inherits n's suffix
		*heir = &node[T]{n.label.rest(common), n.value, n.left, n.right}

		// sibling handles the new suffix
		*sibling = &node[T]{label: label.rest(common), value: value}

		// Truncate n's label
		n.label = n.label.truncated(common)
	}

}

type IPMapBuilder[T any] struct {
	root node[T]
}

type IPMap[T any] struct {
	root node[T]
}

//func (m *IPMapBuilder[T]) Set(ip netip.Addr, value T) error {
//	// TODO: do we care?
//	if !ip.IsValid() {
//		return fmt.Errorf("IP is not valid: %v", ip)
//	}
//
//	m.root.set(ip, value)
//	return nil
//}

func (m *IPMapBuilder[T]) Get(ip netip.Addr) (*T, error) {
	return nil, nil
}

func labelFromPrefix(prefix netip.Prefix) label {
	return label{value: u128From16(prefix.Addr.As16()), len: uint8(prefix.Bits())}
}

func (m *IPMapBuilder[T]) SetPrefix(prefix netip.Prefix, value *T) error {
	// TODO: do we care?
	if !prefix.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", prefix)
	}

	m.root.set(labelFromPrefix(prefix), value)

	return nil
}
