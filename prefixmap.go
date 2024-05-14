package netipmap

import (
	"fmt"
	"net/netip"
)

// label represents one node's key fragment.
// len is the length of the label in bits, counting from the most-significant
// bit toward the least.
// value should not have any bits set beyond len.
type label struct {
	value uint128
	len   uint8
}

func NewLabel(value uint128, len uint8) label {
	return label{value: value.bitsClearedFrom(len), len: len}
}

func (l label) truncated(n uint8) label {
	return label{l.value.bitsClearedFrom(n), n}
}

// rest returns a new label starting from the bit at position from, counting
// from the most-significant bit toward the least.
func (l label) rest(from uint8) label {
	return label{l.value.shiftLeft(from), l.len - from}
}

// Prints the most significant l.len bits of l.value, as hex
func (l label) String() string {
	out := fmt.Sprintf("%0*x", (l.len+3)/4, l.value.hi)
	if l.len > 64 {
		out += fmt.Sprintf("%0*x", (l.len-64+3)/4, l.value.lo)
	}
	out += fmt.Sprintf("/%d", l.len)
	return out
}

// commonPrefixLen returns the length of the common prefix between l and
// other, truncated to the length of the shorter of the two.
func (l label) commonPrefixLen(other label) uint8 {
	common := l.value.commonPrefixLen(other.value)
	// min(l.len, other.len, common)
	if l.len < other.len {
		if l.len < common {
			return l.len
		}
		return common
	} else {
		if other.len < common {
			return other.len
		}
		return common
	}
}

func (l label) isPrefixOf(other label) bool {
	return l.len <= other.len && l.value == other.value.bitsClearedFrom(l.len)
}

// If the shorter of l and other is a prefix of the longer, return the length of
// the longer label. Otherwise, return the length of the common prefix, truncated
// to the length of the shorter label.
func (l label) prefixUnionLen(other label) uint8 {
	if l.len == other.len {
		return l.commonPrefixLen(other)
	} else {
		var shorter, longer label
		if l.len < other.len {
			shorter, longer = l, other
		} else {
			shorter, longer = other, l
		}
		if shorter.isPrefixOf(longer) {
			return longer.len
		}
		return shorter.commonPrefixLen(longer)
	}
}

type node[T any] struct {
	label label
	value T
	left  *node[T]
	right *node[T]
}

func NewNode[T any](label label, value T) *node[T] {
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

func labelFromPrefix(prefix netip.Prefix) label {
	return label{value: u128From16(prefix.Addr().As16()), len: uint8(prefix.Bits())}
}

func (n *node[T]) set(label label, value T) {
	fmt.Println("set", label, value)
	/*if n == nil {
		n = &node[T]{label: label, value: value}
		return
	}*/
	if label == n.label {
		n.value = value
		return
	}

	if n.label.isPrefixOf(label) {
		// n.label is a prefix of the new label, so recurse into the
		// appropriate child of n (or create it).
		var next **node[T]
		if getBit(label.value, n.label.len) {
			next = &n.right
		} else {
			next = &n.left
		}
		if *next == nil {
			*next = &node[T]{label.rest(n.label.len), value, nil, nil}
		} else {
			(*next).set(label.rest(n.label.len), value)
		}
	} else {
		// Split n and create two new children: an 'heir' to inherit n's
		// suffix, and a 'sibling' to handle the new suffix.
		common := n.label.commonPrefixLen(label)
		heir := &node[T]{n.label.rest(common), n.value, n.left, n.right}
		sibling := &node[T]{label.rest(common), value, nil, nil}

		// The bit after the common prefix determines which child will handle
		// which suffix. If n.label has a 0, then left will inherit n.label's
		// suffix, and right will handle the new suffix.
		if getBit(n.label.value, common) {
			n.left = sibling
			n.right = heir
		} else {
			n.left = heir
			n.right = sibling
		}

		// n's label needs to be truncated at the split point
		n.label = n.label.truncated(common)
	}
	n.prettyPrint("", "")
}

func (n *node[T]) prettyPrint(indent string, prefix string) {
	if n == nil {
		return
	}

	fmt.Printf("%s%s%s: %v\n", indent, prefix, n.label, n.value)
	n.left.prettyPrint(indent+"  ", "L:")
	n.right.prettyPrint(indent+"  ", "R:")
}

// have: 0000
// get:  0011 -> nil
// get:  00001 -> nil
//
// have: 00
// -       L:01
// -       R:10
// get:  0001 ->
func (n *node[T]) get(label label) (val T, ok bool) {
	fmt.Println("get", label)
	if label == n.label {
		return n.value, true
	}
	common := n.label.commonPrefixLen(label)

	var next *node[T]
	if getBit(label.value, common) {
		next = n.right
	} else {
		next = n.left
	}
	if next == nil {
		return val, false
	}
	return next.get(label.rest(n.label.len))
}

func (n *node[T]) copy() *node[T] {
	if n == nil {
		return nil
	}

	return &node[T]{
		label: n.label,
		value: n.value,
		left:  n.left.copy(),
		right: n.right.copy(),
	}
}

type PrefixMapBuilder[T any] struct {
	root node[T]
}

type PrefixMap[T any] struct {
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

func (m *PrefixMapBuilder[T]) Set(prefix netip.Prefix, value T) error {
	// TODO: do we care?
	if !prefix.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", prefix)
	}

	m.root.set(labelFromPrefix(prefix), value)

	return nil
}

func (m *PrefixMapBuilder[T]) PrefixMap() *PrefixMap[T] {
	return &PrefixMap[T]{root: *m.root.copy()}
}

func (m *PrefixMap[T]) Get(prefix netip.Prefix) (T, bool) {
	return m.root.get(labelFromPrefix(prefix))
}
