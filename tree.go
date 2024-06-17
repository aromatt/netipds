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

	// Not every node stores a value; some nodes are just shared prefixes
	hasValue bool
}

// newTree returns a new tree with the provided key.
func newTree[T any](k key) *tree[T] {
	return &tree[T]{key: k}
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

// clearValue removes the value from t.
func (t *tree[T]) clearValue() {
	var zeroVal T
	t.value = zeroVal
	t.hasValue = false
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

// setChildrenFrom sets t's children to o's (without copying them) and returns
// t.
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
	var left, right *tree[T]
	if o.left != nil {
		left = o.left.copy()
	}
	if o.right != nil {
		right = o.right.copy()
	}
	return t.setChildren(left, right)
}

// moveChildrenFrom moves o's children to t (removing them from o) and returns
// t.
func (t *tree[T]) moveChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	t.setChildrenFrom(o)
	o.setChildren(nil, nil)
	return t
}

// childPtr returns a pointer to the child corresponding to the bit at offset
// in k (left if 0, right if 1).
func (t *tree[T]) childPtr(k key, offset uint8) **tree[T] {
	if zero, _ := k.hasBitZeroAt(offset); zero {
		return &t.left
	}
	return &t.right
}

// childPtrs returns pointers to the (a) child corresponding to the bit at
// offset in k and (b) its sibling
func (t *tree[T]) childPtrs(k key, offset uint8) (a **tree[T], b **tree[T]) {
	if zero, _ := k.hasBitZeroAt(offset); zero {
		return &t.left, &t.right
	}
	return &t.right, &t.left
}

// setChildFromBit sets t's child to the provided node based on the bit at
// offset i in k, if it isn't already set. A provided nil is ignored.
func (t *tree[T]) setChildFromBit(k key, i uint8, n *tree[T]) *tree[T] {
	childPtr := t.childPtr(k, i)
	if *childPtr == nil && n != nil {
		*childPtr = n
	}
	return t
}

// setChildrenFromBit sets t's children to the provided nodes based on the bit
// at offset i in k, if they aren't already set. Provided nils are ignored.
func (t *tree[T]) setChildrenFromBit(k key, i uint8, a, b *tree[T]) *tree[T] {
	aPtr, bPtr := t.childPtrs(k, i)
	if *aPtr == nil && a != nil {
		*aPtr = a
	}
	if *bPtr == nil && b != nil {
		*bPtr = b
	}
	return t
}

// copy returns a copy of t, creating copies of all descendants of t in the
// process.
func (t *tree[T]) copy() *tree[T] {
	return newTree[T](t.key).copyChildrenFrom(t).setValueFrom(t)
}

func (t *tree[T]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	if hideVal {
		ret = fmt.Sprintf("%s%s%s\n", indent, pre, t.key.StringRel())
	} else {
		ret = fmt.Sprintf("%s%s%s: %v\n", indent, pre, t.key.StringRel(), t.value)
	}
	if t.left != nil {
		ret += t.left.stringImpl(indent+"  ", "L:", hideVal)
	}
	if t.right != nil {
		ret += t.right.stringImpl(indent+"  ", "R:", hideVal)
	}
	return ret
}

func (t *tree[T]) String() string {
	return t.stringImpl("", "", false)
}

// size returns the number of nodes within t that have values.
// TODO: keep track of this instead of calculating it lazily
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

// insert inserts value v at key k with path compression.
func (t *tree[T]) insert(k key, v T) *tree[T] {
	// inserting at t itself
	if t.key.equalFromRoot(k) {
		return t.setValue(v)
	}

	common := t.key.commonPrefixLen(k)
	switch {
	// inserting at a descendant; recurse into the appropriate child
	case common == t.key.len:
		childPtr := t.childPtr(k, t.key.len)
		if *childPtr == nil {
			*childPtr = newTree[T](k.rest(t.key.len)).setValue(v)
		}
		*childPtr = (*childPtr).insert(k, v)
		return t
	// inserting at a prefix of t.key; create a new node at k with t as its
	// sole child
	case common == k.len:
		newNode := newTree[T](k).setValue(v).setChildFromBit(t.key, k.len, t)
		t.key.offset = newNode.key.len
		return newNode
	// k diverges in the middle of t.key; create a new parent at their common
	// prefix with children k and t
	case common < t.key.len:
		newParent := newTree[T](t.key.truncated(common))
		kChild := newTree[T](k.rest(common)).setValue(v)
		newParent.setChildrenFromBit(t.key, common, t, kChild)
		t.key.offset = common
		return newParent
	// nothing to do
	default:
		return t
	}
}

// insertLazy inserts value v at key k without path compression.
func (t *tree[T]) insertLazy(k key, v T) *tree[T] {
	switch {
	// inserting at t itself
	case t.key.equalFromRoot(k):
		return t.setValue(v)
	// inserting at a descendant
	case t.key.commonPrefixLen(k) == t.key.len:
		if zero, _ := k.hasBitZeroAt(t.key.len); zero {
			if t.left == nil {
				t.left = newTree[T](t.key.left())
			}
			t.left = t.left.insertLazy(k, v)
		} else {
			if t.right == nil {
				t.right = newTree[T](t.key.right())
			}
			t.right = t.right.insertLazy(k, v)
		}
		return t
	// nothing to do
	default:
		return t
	}
}

// compress performs path compression on tree t.
func (t *tree[T]) compress() *tree[T] {
	switch {
	case t.left == nil && t.right == nil:
		return t
	case t.left == nil:
		t.right.key.offset = t.key.offset
		return t.right
	case t.right == nil:
		t.left.key.offset = t.key.offset
		return t.left
	default:
		return t
	}
}

// remove removes the exact provided key from the tree, if it exists, and
// performs path compression.
func (t *tree[T]) remove(k key) *tree[T] {
	switch {
	// removing t itself
	case k.equalFromRoot(t.key):
		if t.hasValue {
			t.clearValue()
		}
		switch {
		// no children (deleting a leaf node)
		case t.left == nil && t.right == nil:
			return nil
		// only one child; merge with it
		case t.left == nil:
			t.right.key.offset = t.key.offset
			return t.right
		case t.right == nil:
			t.left.key.offset = t.key.offset
			return t.left
		// t is a shared prefix node, so it needs to persist
		default:
			return t
		}
	// removing a descendant of t; recurse into the appropriate child
	case t.key.isPrefixOf(k):
		childPtr := t.childPtr(k, t.key.len)
		if *childPtr != nil {
			*childPtr = (*childPtr).remove(k)
		}
		return t
	// nothing to do
	default:
		return t
	}
}

// subtractKey removes k and all of its descendants from the tree, leaving the
// remaining key space behind. If k is a descendant of t, then new nodes may be
// created to inherit t's value, filling in the gaps around k.
func (t *tree[T]) subtractKey(k key) *tree[T] {
	// this whole branch is being subtracted; no need to traverse further
	if t.key.equalFromRoot(k) || k.isPrefixOf(t.key) {
		return nil
	}
	// a child of t is being subtracted
	if t.key.isPrefixOf(k) {
		childPtr := t.childPtr(k, t.key.len)
		if *childPtr != nil {
			*childPtr = (*childPtr).subtractKey(k.rest(t.key.len))
		} else {
			t.insertHole(k, t.value)
		}
		if t.right == nil && t.left == nil && !t.hasValue {
			return nil
		}
	}
	return t
}

// subtractTree removes all value-bearing nodes from t that are also
// value-bearing nodes found in o.
func (t *tree[T]) subtractTree(o tree[T]) *tree[T] {
	if o.hasValue {
		// this whole branch is being subtracted; no need to traverse further
		if t.key.equalFromRoot(o.key) || o.key.isPrefixOf(t.key) {
			return nil
		}
		// a child of t is being subtracted
		if t.key.isPrefixOf(o.key) {
			t.insertHole(o.key, t.value)
		}
	}
	// traverse children of both t and o as able
	if o.left != nil {
		if t.left != nil {
			t.left = t.left.subtractTree(*o.left)
		} else {
			t = t.subtractTree(*o.left)
		}
	}
	if o.right != nil {
		if t.right != nil {
			t.right = t.right.subtractTree(*o.right)
		} else {
			t = t.subtractTree(*o.right)
		}
	}
	return t
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *tree[T]) insertHole(k key, v T) *tree[T] {
	switch {
	// removing t itself (no descendants will receive v)
	case t.key.equalFromRoot(k):
		return nil
	// k is a descendant of t; start digging a hole to k
	case t.key.isPrefixOf(k):
		t.clearValue()
		// recurse to appropriate child and create a sibling to receive v
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
	// nothing to do
	default:
		return t
	}
}

// walk traverses the tree starting at this tree's root, following the
// provided path and calling fn(node) at each visited node.
//
// When the provided path is exhausted, walk continues by visiting all
// children.
//
// If fn returns true, then walk stops traversing any deeper.
func (t *tree[T]) walk(path key, fn func(*tree[T]) bool) {
	// Never call fn on root node
	if !t.key.isZero() {
		if fn(t) {
			return
		}
	}

	zero, anyPathLeft := path.hasBitZeroAt(t.key.commonPrefixLen(path))

	if (zero || !anyPathLeft) && t.left != nil {
		t.left.walk(path.rest(t.key.len), fn)
	}
	if (!zero || !anyPathLeft) && t.right != nil {
		t.right.walk(path.rest(t.key.len), fn)
	}
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T]) get(k key) (val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.key.len >= k.len {
			if n.key.equalFromRoot(k) && n.hasValue {
				val, ok = n.value, true
			}
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
		ret = n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue
		if ret {
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
// If strict == true, the key itself is not considered.
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
// tree, unless strict == true. descendantsOf returns an empty tree if the
// provided key is not in the tree.
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
// strict == true. ancestorsOf returns an empty tree if key has no ancestors in
// the tree.
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
//
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

// filterCopy returns a recursive copy of t that includes only keys that are
// encompassed by o.
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
// TODO: is the returned tree fully compressed?
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

// overlapsKey reports whether any key in t overlaps k.
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
