package netipds

import (
	"fmt"
)

// tree is a binary radix tree with path compression.
type tree[T any] struct {
	key   key
	value T
	left  *tree[T]
	right *tree[T]

	// Not every node has a value. A node may be just a shared prefix.
	hasValue bool
}

// newTree returns a new tree with the provided key.
func newTree[T any](k key) *tree[T] {
	return &tree[T]{key: k}
}

// clearValue removes the value from t.
func (t *tree[T]) clearValue() {
	var zeroVal T
	t.value = zeroVal
	t.hasValue = false
}

// setKey sets t's key to k and returns t.
func (t *tree[T]) setKey(k key) *tree[T] {
	t.key = k
	return t
}

// setValue sets t's value to v and returns t.
func (t *tree[T]) setValue(v T) *tree[T] {
	t.value = v
	t.hasValue = true
	return t
}

// setValueFrom sets t's value to o's value and returns t.
func (t *tree[T]) setValueFrom(o *tree[T]) *tree[T] {
	if o != nil && o.hasValue {
		return t.setValue(o.value)
	}
	return t
}

// moveValueFrom moves o's value to t (removing it from o) and returns t.
func (t *tree[T]) moveValueFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	if o.hasValue {
		t.value, t.hasValue = o.value, true
		o.clearValue()
	}
	return t
}

// setChildren sets t's children to the provided left and right trees and
// returns t.
func (t *tree[T]) setChildren(left *tree[T], right *tree[T]) *tree[T] {
	t.left = left
	t.right = right
	return t
}

// setChildrenFrom sets t's children to o's (without copying them) and returns t.
func (t *tree[T]) setChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	t = t.setChildren(o.left, o.right)
	return t
}

// copyChildrenFrom sets t's children to copies of o's children and returns t.
func (t *tree[T]) copyChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	return t.setChildren(o.left.copy(), o.right.copy())
}

// moveChildrenFrom moves o's children to t (removing them from o) and returns t.
func (t *tree[T]) moveChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	t.setChildrenFrom(o)
	o.setChildren(nil, nil)
	return t
}

// copy returns a copy of t, creating copies of all descendants of t in the
// process. If t is nil, copy returns nil.
func (t *tree[T]) copy() *tree[T] {
	if t == nil {
		return nil
	}
	return newTree[T](t.key).copyChildrenFrom(t).setValueFrom(t)
}

// isZero returns true if this node's key is the zero key.
// TODO: change name to isRoot?
func (t *tree[T]) isZero() bool {
	return t.key.isZero()
}

// prettyPrint prints the tree in a human-readable format.
func (t *tree[T]) stringHelper(indent string, prefix string, hideValue bool) string {
	ret := fmt.Sprintf("%s%s%s: %v\n", indent, prefix, t.key.StringRelative(), t.value)
	if t.left != nil {
		ret += t.left.stringHelper(indent+"  ", "L:", hideValue)
	}
	if t.right != nil {
		ret += t.right.stringHelper(indent+"  ", "R:", hideValue)
	}
	return ret
}

func (t *tree[T]) String() string {
	return t.stringHelper("", "", false)
}

func (t *tree[T]) size() int {
	size := 0
	if t.hasValue {
		size = 1
	}
	if t.left != nil {
		size += t.left.size()
	}
	if t.right != nil {
		size += t.right.size()
	}
	return size
}

func (t *tree[T]) insert(k key, v T) *tree[T] {
	common := t.key.commonPrefixLen(k)
	switch {
	case t.key == k:
		return t.setValue(v)
	case common == t.key.len:
		return t.insertChild(k, v)
	case common == k.len:
		return t.insertParent(k, v)
	case common < t.key.len:
		return t.insertFork(k, v, common)
	default:
		// TODO
		panic("unreachable")
	}
}

// insertChild inserts or updates the appropriate child of t for key k.
func (t *tree[T]) insertChild(k key, v T) *tree[T] {
	var next **tree[T]
	if zero, _ := k.hasBitZeroAt(t.key.len); zero {
		next = &t.left
	} else {
		next = &t.right
	}
	if *next == nil {
		*next = newTree[T](k.rest(t.key.len)).setValue(v)
	} else {
		*next = (*next).insert(k, v)
	}
	return t
}

// insertParent inserts and returns a new node with t as its sole child.
func (t *tree[T]) insertParent(k key, v T) *tree[T] {
	newNode := newTree[T](k).setValue(v)
	if zero, _ := t.key.hasBitZeroAt(k.len); zero {
		newNode.left = t
	} else {
		newNode.right = t
	}
	t.key.offset = newNode.key.len
	return newNode
}

// insertFork inserts a new node at the common prefix of t.key and k
// with value v and t.key and k as children, and returns the new node.
func (t *tree[T]) insertFork(k key, v T, common uint8) *tree[T] {
	parent := newTree[T](t.key.truncated(common))
	t.key.offset = common
	sibling := newTree[T](k.rest(common)).setValue(v)
	if zero, _ := k.hasBitZeroAt(common); zero {
		parent.left = sibling
		parent.right = t
	} else {
		parent.left = t
		parent.right = sibling
	}
	return parent
}

// remove removes the exact key provided from the tree, if it exists.
func (t *tree[T]) remove(k key) *tree[T] {
	if k.equalFromRoot(t.key) {
		if t.hasValue {
			t.clearValue()
		}
		switch {
		// Deleting a leaf node; no children to worry about
		case t.left == nil && t.right == nil:
			return nil
		// If there is only one child, merge with it.
		case t.left == nil:
			t.right.key.offset = t.key.offset
			return t.right
		case t.right == nil:
			t.left.key.offset = t.key.offset
			return t.left
		// This is a shared prefix node, so it needs to persist.
		default:
			return t
		}
	}

	// t.key is a prefix of the key to remove, so recurse into the appropriate
	// child of t.
	if t.key.isPrefixOf(k) {
		if zero, _ := k.hasBitZeroAt(t.key.len); zero {
			if t.left != nil {
				t.left = t.left.remove(k.rest(t.key.len))
			}
		} else {
			if t.right != nil {
				t.right = t.right.remove(k.rest(t.key.len))
			}
		}
	}

	return t
}

// subtract removes the key and all of its descendants from the tree, leaving
// the remaining key space behind. New nodes may be created in the process.
func (t *tree[T]) subtract(k key) *tree[T] {
	common := t.key.commonPrefixLen(k)
	switch {
	case t.key.equalFromRoot(k):
		return nil
	case common == 0:
		return t.subtractChild(k)
	case common == t.key.len:
		return t.insertHole(k, t.value)
	case common == k.len:
		return nil
	case common < t.key.len:
		return t
	default:
		// TODO
		panic("unreachable")
	}
}

func (t *tree[T]) subtractChild(k key) *tree[T] {
	if zero, _ := k.hasBitZeroAt(t.key.len); zero {
		if t.left != nil {
			t.left = t.left.subtract(k.rest(t.key.len))
		}
	} else {
		if t.right != nil {
			t.right = t.right.subtract(k.rest(t.key.len))
		}
	}
	return t
}

func (t *tree[T]) insertHole(k key, v T) *tree[T] {
	switch {
	case t.key.equalFromRoot(k):
		return nil
	case t.key.isPrefixOf(k):
		t.clearValue()
		if zero, _ := k.hasBitZeroAt(t.key.len); zero {
			if t.right == nil {
				t.right = newTree[T](t.key.right()).setValue(v)
			}
			t.left = newTree[T](t.key.left()).insertHole(k, v)
		} else {
			if t.left == nil {
				t.left = newTree[T](t.key.left()).setValue(v)
			}
			t.right = newTree[T](t.key.right()).insertHole(k, v)
		}
		return t
	default:
		return t
	}
}

// walk traverses the tree starting at this tree's root, following the
// provided path and calling fn(node) at each visited node.
//
// The return value of fn is a boolean indicating whether traversal should
// stop.
//
// If path is the zero key, all descendants of this tree are visited.
func (t *tree[T]) walk(path key, fn func(*tree[T]) bool) {
	// Never call fn on root node
	if !t.isZero() {
		if fn(t) {
			return
		}
	}

	nextPath := path.rest(t.key.len)
	zero, ok := path.hasBitZeroAt(t.key.commonPrefixLen(path))

	// !ok means we've navigated to the end of the path constraint. Visit all
	// children from here on.
	if !ok {
		if t.left != nil {
			t.left.walk(nextPath, fn)
		}
		if t.right != nil {
			t.right.walk(nextPath, fn)
		}
		return
	}

	// Visit the child that matches the next bit in the path.
	switch zero {
	case true:
		if t.left != nil {
			t.left.walk(nextPath, fn)
		}
	case false:
		if t.right != nil {
			t.right.walk(nextPath, fn)
		}
	}
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T]) get(k key) (val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.key.len >= k.len {
			if n.key.equalFromRoot(k) && n.hasValue {
				val, ok = n.value, true
			}
			// Always stop traversal if we've reached the end of the path.
			return true
		}
		return false
	})
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *tree[T]) contains(k key) (ret bool) {
	t.walk(k, func(n *tree[T]) bool {
		if ret = (n.key.equalFromRoot(k) && n.hasValue); ret {
			return true
		}
		return false
	})
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
func (t *tree[T]) encompasses(k key, strict bool) (ret bool) {
	t.walk(k, func(n *tree[T]) bool {
		if ret = (n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue); ret {
			return true
		}
		return false
	})
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T]) rootOf(k key, strict bool) (outKey key, val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue {
			outKey, val, ok = n.key, n.value, true
			return true
		}
		return false
	})
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict is true, the key itself is not considered.
func (t *tree[T]) parentOf(k key, strict bool) (outKey key, val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue {
			outKey, val, ok = n.key, n.value, true
		}
		return false
	})
	return
}

// descendantsOf returns the sub-tree containing all descendants of the
// provided key. The key itself will be included if it has an entry in the
// tree, unless strict. descendantsOf returns the empty tree if the provided
// key is not in the tree.
func (t *tree[T]) descendantsOf(k key, strict bool) (ret *tree[T]) {
	ret = &tree[T]{}
	t.walk(k, func(n *tree[T]) bool {
		if k.isPrefixOf(n.key) {
			ret = ret.setKey(n.key.rooted()).setValueFrom(n).setChildrenFrom(n)
			return true
		}
		return false
	})
	return
}

// ancestorsOf returns the sub-tree containing all ancestors of the provided
// key. The key itself will be included if it has an entry in the tree, unless
// strict. ancestorsOf returns an empty tree if key has no ancestors in the
// tree.
func (t *tree[T]) ancestorsOf(k key, strict bool) (ret *tree[T]) {
	ret = &tree[T]{}
	t.walk(k, func(n *tree[T]) bool {
		if !n.key.isPrefixOf(k) {
			return true
		}
		if n.hasValue {
			ret.insert(n.key, n.value)
		}
		return false
	})
	return
}

// filter updates t to include only the keys encompassed by o.
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
func (t *tree[T]) filter(o tree[bool]) {
	remove := make([]key, 0)
	t.walk(key{}, func(n *tree[T]) bool {
		if !o.encompasses(n.key, false) {
			remove = append(remove, n.key)
		}
		return false
	})
	for _, k := range remove {
		t.remove(k)
	}
}

// filterCopy returns a new tree containing all entries of t that are
// encompassed by o.
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
func (t *tree[T]) filterCopy(o tree[bool]) *tree[T] {
	ret := &tree[T]{}
	t.walk(key{}, func(n *tree[T]) bool {
		if n.hasValue && o.encompasses(n.key, false) {
			ret = ret.insert(n.key, n.value)
		}
		return false
	})
	return ret
}

func (t *tree[T]) overlapsKey(k key) bool {
	var ret bool
	t.walk(k, func(n *tree[T]) bool {
		if !n.hasValue {
			return false
		}
		if n.key.isPrefixOf(k) || k.isPrefixOf(n.key) {
			ret = true
			return true
		}
		return false
	})
	return ret
}
