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
		one, ok := l.getBit(n.label.len)
		if !ok {
			// n.label is a prefix of l, so this should never happen
			panic("unexpected end of label")
		}
		if one {
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
		// which suffix.
		// TODO check ok
		if one, _ := n.label.getBit(common); one {
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

// walkPath traverses the tree starting at node, following the provided path and
// calling fn at each visited node.
//
// The arguments to fn are (1) a label containing the prefix accumulated during
// the traversal up until the current node and (2) the current node.
//
// The return values of fn are (1) a boolean indicating whether traversal
// should stop and (2) an error. If fn returns true, traversal stops and
// walkPath returns nil.
//
// If path is the zero label, all descendants of n are visited.
func (n *node[T]) walk(
	path label,
	pre label,
	fn func(label, *node[T]) (bool, error),
) error {
	if n == nil {
		return nil
	}

	// Never call fn on root node
	if !n.isZero() {
		stop, err := fn(pre, n)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}

	nextPath := path.rest(n.label.len)
	nextPre := pre.concat(n.label)
	one, ok := path.getBit(n.label.commonPrefixLen(path))

	// Visit the child that matches the next bit in the path. If the path is
	// exhausted (i.e. !ok), visit both children.
	var err error
	if !one || !ok {
		if err = n.left.walk(nextPath, nextPre, fn); err != nil {
			return err
		}
	}
	if one || !ok {
		if err = n.right.walk(nextPath, nextPre, fn); err != nil {
			return err
		}
	}
	return nil
}

// get returns the value associated with the exact label provided, if it exists.
func (n *node[T]) get(l label) (val T, ok bool) {
	n.walk(l, label{}, func(pre label, m *node[T]) (bool, error) {
		// If the label matches, stop traversing and return the value
		if m.hasValue && pre.concat(m.label) == l {
			val, ok = m.value, true
			return true, nil
		}
		return false, nil
	})
	return val, ok
}

// getDescendants returns a map of all descendants of the provided label and
// their associated values.
func (n *node[T]) getDescendants(l label) map[label]T {
	fmt.Println("getDescendants", l)
	descendants := make(map[label]T)
	n.walk(l, label{}, func(pre label, m *node[T]) (bool, error) {
		if m.hasValue && pre.concat(m.label).isPrefixOf(l) {
			descendants[pre.concat(m.label)] = m.value
		}
		return false, nil
	})
	return descendants
}
