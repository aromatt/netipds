package netipmap

import (
	"fmt"
)

type node[T any] struct {
	label label
	value T
	left  *node[T]
	right *node[T]

	// Not every node has a value. A node may be just a shared prefix.
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

func (n *node[T]) moveValueFrom(m *node[T]) *node[T] {
	if m == nil {
		return n
	}
	if m.hasValue {
		var zero T
		n.value, n.hasValue = m.value, true
		m.value, m.hasValue = zero, false
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
		zero, ok := l.isBitZero(n.label.len)
		if !ok {
			// n.label is a prefix of l, so this should never happen
			panic("unexpected end of label")
		}
		if zero {
			next = &n.left
		} else {
			next = &n.right
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
		heir := newNode[T](n.label.rest(common)).moveValueFrom(n).moveChildrenFrom(n)
		sibling := newNode[T](l.rest(common)).withValue(value)

		// The bit after the common prefix determines which child will handle
		// which suffix.
		// TODO check ok
		if zero, _ := n.label.isBitZero(common); zero {
			n.left = heir
			n.right = sibling
		} else {
			n.left = sibling
			n.right = heir
		}

		// n's label needs to be truncated at the split point
		n.label = n.label.truncated(common)
	}
}

// walkPath traverses the tree starting at node, following the provided path and
// calling fn at each visited node.
//
// The arguments to fn are (1) a label containing the prefix accumulated during
// the traversal including the current node and (2) the current node.
//
// The return value of fn is a boolean indicating whether traversal
// should stop.
//
// If path is the zero label, all descendants of n are visited.
func (n *node[T]) walk(
	path label,
	pre label,
	fn func(label, *node[T]) bool,
) {
	if n == nil {
		return
	}

	// Never call fn on root node
	if !n.isZero() {
		if fn(pre.concat(n.label), n) {
			return
		}
	}

	nextPath := path.rest(n.label.len)
	nextPre := pre.concat(n.label)
	zero, ok := path.isBitZero(n.label.commonPrefixLen(path))

	// Visit the child that matches the next bit in the path. If the path is
	// exhausted (i.e. !ok), visit both children.
	if zero || !ok {
		n.left.walk(nextPath, nextPre, fn)
	}
	if !zero || !ok {
		n.right.walk(nextPath, nextPre, fn)
	}
	return
}

// get returns the value associated with the exact label provided, if it exists.
func (n *node[T]) get(l label) (val T, ok bool) {
	n.walk(l, label{}, func(key label, m *node[T]) bool {
		if key == l && m.hasValue {
			val, ok = m.value, true
			return true
		}
		return false
	})
	return val, ok
}

// walkDescendants calls fn on each descendant of the provided label, including
// itself.
func (n *node[T]) walkDescendants(l label, fn func(label, *node[T])) {
	n.walk(l, label{}, func(key label, m *node[T]) bool {
		if l.isPrefixOf(key) && m.hasValue {
			fn(key, m)
		}
		return false
	})
}

// walkAncestors calls fn on each ancestor of the provided label, including
// itself.
func (n *node[T]) walkAncestors(l label, fn func(label, *node[T])) {
	n.walk(l, label{}, func(key label, m *node[T]) bool {
		if key.isPrefixOf(l) && m.hasValue {
			fn(key, m)
		}
		return false
	})
}
