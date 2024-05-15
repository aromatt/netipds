package netipmap

import (
	"fmt"
	"net/netip"
)

// getBit returns true if bit at position i is set in u, counting from the
// most-significant bit toward the least.
func getBit(u uint128, i uint8) bool {
	if i < 64 {
		return u.hi&(uint64(1)<<(63-i)) > 0
	}
	return u.lo&(uint64(1)<<(127-i)) > 0
}

// label represents one node's key fragment.
// len is the length of the label in bits, counting from the most-significant
// bit toward the least.
// value should not have any bits set beyond len.
type label struct {
	value uint128
	len   uint8
}

func newLabel(value uint128, len uint8) label {
	return label{value: value.bitsClearedFrom(len), len: len}
}

func labelFromPrefix(prefix netip.Prefix) label {
	return newLabel(u128From16(prefix.Addr().As16()), uint8(prefix.Bits()))
}

// bitsClearedFrom returns a copy of label truncated to n bits.
func (l label) truncated(n uint8) label {
	return newLabel(l.value, n)
}

// rest returns a new label starting from the bit at position from, counting
// from the most-significant bit toward the least.
func (l label) rest(from uint8) label {
	return newLabel(l.value.shiftLeft(from), l.len-from)
}

// concat returns a new label with the bits of other appended to l.
func (l label) concat(other label) label {
	return newLabel(l.value.or(other.value.shiftLeft(l.len)), l.len+other.len)
}

// Prints the least significant bits of l.value as hex, followed by "/len".
func (l label) String() string {
	var ret string
	just := l.value.shiftRight(128 - l.len)
	if l.value.hi == 0 && l.value.lo == 0 {
		ret = "0"
	} else {
		if just.hi > 0 {
			ret = fmt.Sprintf("%x", just.hi)
		}
		if just.lo > 0 {
			if just.hi > 0 {
				ret = fmt.Sprintf("%s%0*x", ret, (l.len-64)/4, just.lo)
			} else {
				ret = fmt.Sprintf("%s%x", ret, just.lo)
			}
		}
	}
	return fmt.Sprintf("%s/%d", ret, l.len)
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

	// Not every node has a value. A node may by just a shared prefix.
	hasValue bool
}

func newNode[T any](l label) *node[T] {
	return &node[T]{label: l}
}

func (n *node[T]) withValue(value T) *node[T] {
	n.value = value
	n.hasValue = true
	return n
}

func (n *node[T]) withChildren(left *node[T], right *node[T]) *node[T] {
	n.left = left
	n.right = right
	return n
}

func (n *node[T]) withChildrenFrom(other *node[T]) *node[T] {
	if other == nil {
		return n
	}
	return n.withChildren(other.left.copy(), other.right.copy())
}

func (n *node[T]) withValueFrom(other *node[T]) *node[T] {
	if other == nil {
		return n
	}
	if other.hasValue {
		return n.withValue(other.value)
	}
	return n
}

func (n *node[T]) set(l label, value T) {
	fmt.Println("set", l, value)
	/*if n == nil {
		n = &node[T]{label: label, value: value}
		return
	}*/
	if l == n.label {
		n.value = value
		n.hasValue = true
		return
	}

	if n.label.isPrefixOf(l) {
		// n.label is a prefix of the new label, so recurse into the
		// appropriate child of n (or create it).
		var next **node[T]
		if getBit(l.value, n.label.len) {
			next = &n.right
		} else {
			next = &n.left
		}
		if *next == nil {
			*next = newNode[T](l.rest(n.label.len)).withValue(value)
		} else {
			(*next).set(l.rest(n.label.len), value)
		}
	} else {
		common := n.label.commonPrefixLen(l)

		// Split n and create two new children: an "heir" to inherit n's
		// suffix, and a sibling to handle the new suffix.
		heir := newNode[T](n.label.rest(common)).withChildrenFrom(n).withValueFrom(n)
		sibling := newNode[T](l.rest(common)).withValue(value)

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

	fmt.Printf("%s%s%s: %v %v\n", indent, prefix, n.label, n.value, n.hasValue)
	n.left.prettyPrint(indent+"  ", "L:")
	n.right.prettyPrint(indent+"  ", "R:")
}

// have: 00
// -       01
// -       10
// get:  00
func (n *node[T]) walkLabel(
	l label,
	pre label,
	fn func(label, *node[T]) error,
) error {
	fmt.Println("traverse", l)
	if err := fn(pre, n); err != nil {
		return err
	}
	common := n.label.commonPrefixLen(l)
	var next *node[T]
	if getBit(l.value, common) {
		fmt.Println("going right")
		next = n.right
	} else {
		fmt.Println("going left")
		next = n.left
	}
	if next != nil {
		return next.walkLabel(l.rest(n.label.len), pre.concat(n.label), fn)
	}
	return nil
}

func (n *node[T]) get(l label) (val T, ok bool) {
	n.walkLabel(l, label{}, func(pre label, m *node[T]) error {
		fmt.Printf("walking\tpre: %s\tfull: %s\n", pre, pre.concat(m.label))
		if m.hasValue && pre.concat(m.label) == l {
			val = m.value
			ok = true
		}
		return nil
	})
	return val, ok
}

//func (n *node[T]) getDescendants(l label) (map[label]T, error) {
//	descendants := make(map[label]T)
//	n.traverse(l, label{}, func(pre label, node *node[T]) error {
//	})
//	return descendants, nil
//
//}

func (n *node[T]) copy() *node[T] {
	if n == nil {
		return nil
	}
	return newNode[T](n.label).withValueFrom(n).withChildrenFrom(n)
}

type PrefixMapBuilder[T any] struct {
	root node[T]
}

type PrefixMap[T any] struct {
	root node[T]
}

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

// GetDescendants returns all descendants of prefic found in m as a map of Prefixes to values.
//func (m *PrefixMap[T]) GetDescendants(prefix netip.Prefix) (map[netip.Prefix]T, bool) {
//}
