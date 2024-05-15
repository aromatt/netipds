package netipmap

import (
	"fmt"
	"net/netip"
)

// label represents one node's key fragment. The content of the label occupies
// the most-significant len bits of the value field. value should not have any
// bits set beyond len (using newLabel() enforces this).
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

func labelFromString(s string) (label, error) {
	l := label{}
	var valueStr, lenStr string
	if _, err := fmt.Sscanf(s, "%s/%s", &valueStr, &lenStr); err != nil {
		return label{}, fmt.Errorf("failed to parse label: %w", err)
	}
	if _, err := fmt.Sscanf(lenStr, "%d", &l.len); err != nil {
		return label{}, fmt.Errorf("failed to parse label: %w", err)
	}
	if _, err := fmt.Sscanf(valueStr, "%x", &l.value.hi); err != nil {
		return label{}, fmt.Errorf("failed to parse label: %w", err)
	}
	if len(valueStr) > 16 {
		if _, err := fmt.Sscanf(valueStr[len(valueStr)-16:], "%x", &l.value.lo); err != nil {
			return label{}, fmt.Errorf("failed to parse label: %w", err)
		}
	}
	return l, nil
}

// Prints l.value as hex, followed by "/len". The least significant bit in the
// output is the bit at position l.len. Leading zeros are omitted. Examples:
//
//   - label{uint128{0, 1}, 128} => "1/128"
//   - label{uint128{0, 2}, 128} => "2/128"
//   - label{uint128{0, 2}, 127} => "1/127"
//   - label{uint128{1, 1}, 128} => "10000000000000001/128"
//   - label{uint128{1, 0}, 64}  => "1/64"
//   - label{uint128{256, 0}, 56} => "1/56"
//   - label{uint128{256, 0}, 64} => "100/64"
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

// bitsClearedFrom returns a copy of label truncated to n bits.
func (l label) truncated(n uint8) label {
	return newLabel(l.value, n)
}

// rest returns a copy of l starting at the bit at position i.
func (l label) rest(i uint8) label {
	return newLabel(l.value.shiftLeft(i), l.len-i)
}

// getBit returns True if the label's bit at position i is set.
func (l label) getBit(i uint8) bool {
	return l.value.isBitSet(i)
}

// concat returns a new label with the bits of m appended to l.
func (l label) concat(m label) label {
	return newLabel(l.value.or(m.value.shiftRight(l.len)), l.len+m.len)
}

// commonPrefixLen returns the length of the common prefix between l and
// m, truncated to the length of the shorter of the two.
func (l label) commonPrefixLen(m label) uint8 {
	common := l.value.commonPrefixLen(m.value)
	// min(l.len, m.len, common)
	if l.len < m.len {
		if l.len < common {
			return l.len
		}
		return common
	} else {
		if m.len < common {
			return m.len
		}
		return common
	}
}

func (l label) isPrefixOf(m label) bool {
	return l.len <= m.len && l.value == m.value.bitsClearedFrom(l.len)
}

func (l label) isZero() bool {
	return l == label{}
}

// If the shorter of l and m is a prefix of the longer, return the length of
// the longer label. Otherwise, return the length of the common prefix,
// truncated to the length of the shorter label.
func (l label) prefixUnionLen(m label) uint8 {
	if l.len == m.len {
		return l.commonPrefixLen(m)
	} else {
		var shorter, longer label
		if l.len < m.len {
			shorter, longer = l, m
		} else {
			shorter, longer = m, l
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

// withValue sets n's value to v and returns n.
func (n *node[T]) withValue(v T) *node[T] {
	n.value = v
	n.hasValue = true
	return n
}

// withValueFrom sets n's value to m's value and returns n.
func (n *node[T]) withValueFrom(m *node[T]) *node[T] {
	if m == nil {
		return n
	}
	if m.hasValue {
		return n.withValue(m.value)
	}
	return n
}

// withChildren sets n's children to the provided left and right nodes and
// returns n.
func (n *node[T]) withChildren(left *node[T], right *node[T]) *node[T] {
	n.left = left
	n.right = right
	return n
}

// copyChildrenFrom sets n's children to copies of m's children and returns n.
func (n *node[T]) copyChildrenFrom(m *node[T]) *node[T] {
	if m == nil {
		return n
	}
	return n.withChildren(m.left.copy(), m.right.copy())
}

// moveChildrenFrom moves m's children to n (removing them from m) and returns n.
func (n *node[T]) moveChildrenFrom(m *node[T]) *node[T] {
	if m == nil {
		return n
	}
	n, _ = n.withChildren(m.left, m.right), m.withChildren(nil, nil)
	return n
}

// copy returns a copy of n, creating copies of all descendants of n in the
// process.
func (n *node[T]) copy() *node[T] {
	if n == nil {
		return nil
	}
	return newNode[T](n.label).copyChildrenFrom(n).withValueFrom(n)
}

func (n *node[T]) isZero() bool {
	return n.label.isZero()
}

func (n *node[T]) prettyPrint(indent string, prefix string) {
	if n == nil {
		return
	}

	fmt.Printf("%s%s%s: %v\n", indent, prefix, n.label, n.value)
	n.left.prettyPrint(indent+"  ", "L:")
	n.right.prettyPrint(indent+"  ", "R:")
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
		if l.getBit(n.label.len) {
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
		heir := newNode[T](n.label.rest(common)).withValueFrom(n).moveChildrenFrom(n)
		sibling := newNode[T](l.rest(common)).withValue(value)

		// The bit after the common prefix determines which child will handle
		// which suffix. If n.label has a 0, then left will inherit n.label's
		// suffix, and right will handle the new suffix.
		if n.label.getBit(common) {
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

func (n *node[T]) walk(path label, pre label, fn func(label, *node[T]) error) error {
	fmt.Println("walking", n.label)
	if !n.isZero() {
		fmt.Println("calling fn", pre, n.label)
		if err := fn(pre, n); err != nil {
			return err
		}
	} else {
		fmt.Println("skipping zero node")
	}
	common := n.label.commonPrefixLen(path)
	var next *node[T]
	if path.getBit(common) {
		next = n.right
	} else {
		next = n.left
	}
	if next != nil {
		return next.walk(path.rest(n.label.len), pre.concat(n.label), fn)
	}
	return nil
}

func (n *node[T]) get(l label) (val T, ok bool) {
	n.walk(l, label{}, func(pre label, m *node[T]) error {
		fmt.Println("pre.concat(m.label)", pre.concat(m.label))
		if m.hasValue && pre.concat(m.label) == l {
			val = m.value
			ok = true
		}
		return nil
	})
	return
}

//func (n *node[T]) getDescendants(l label) (map[label]T, error) {
//	descendants := make(map[label]T)
//	n.traverse(l, label{}, func(pre label, node *node[T]) error {
//	})
//	return descendants, nil
//
//}

type PrefixMapBuilder[T any] struct {
	root node[T]
}

type PrefixMap[T any] struct {
	root node[T]
}

func (m *PrefixMapBuilder[T]) Set(prefix netip.Prefix, value T) error {
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
