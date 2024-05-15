package netipmap

import (
	"fmt"
)

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
	//n.prettyPrint("", "")
}

func (n *node[T]) walk(path label, pre label, fn func(label, *node[T]) error) error {
	if !n.isZero() {
		if err := fn(pre, n); err != nil {
			return err
		}
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
