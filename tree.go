package netipds

import (
	"fmt"
	"strings"
)

// tree is a binary radix tree with path compression.
type tree[T any] struct {
	key   key
	value T
	left  *tree[T]
	right *tree[T]

	// Not every node stores a value; some nodes are just shared prefixes
	hasEntry bool
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
	t.hasEntry = true
	return t
}

// clearValue removes the value from t.
func (t *tree[T]) clearValue() {
	var zeroVal T
	t.value = zeroVal
	t.hasEntry = false
}

// setValueFrom sets t's value to o's value and returns t.
func (t *tree[T]) setValueFrom(o *tree[T]) *tree[T] {
	if o != nil && o.hasEntry {
		return t.setValue(o.value)
	}
	return t
}

// moveValueFrom moves o's value to t (removing it from o) and returns t.
func (t *tree[T]) moveValueFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	if o.hasEntry {
		t.value, t.hasEntry = o.value, true
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

// child returns a pointer to the specified child of t.
func (t *tree[T]) child(right bool) **tree[T] {
	if right {
		return &t.right
	}
	return &t.left
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

// copy returns a copy of t, creating copies of all of t's descendants in the
// process.
func (t *tree[T]) copy() *tree[T] {
	return newTree[T](t.key).copyChildrenFrom(t).setValueFrom(t)
}

func (t *tree[T]) stringImpl(indent string, pre string, hideVal bool) string {
	//go:coverage ignore
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
	//go:coverage ignore
	return t.stringImpl("", "", false)
}

// size returns the number of nodes within t that have values.
// TODO: keep track of this instead of calculating it lazily
func (t *tree[T]) size() int {
	size := 0
	if t.hasEntry {
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
		if t.hasEntry {
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
		// t is a shared prefix node, so it can't be removed
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
// created to fill in the gaps around k.
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
		if t.right == nil && t.left == nil && !t.hasEntry {
			return nil
		}
	}
	return t
}

// subtractTree removes all entries from t that have counterparts in o. If a
// child of t is removed, then new nodes may be created to fill in the gaps
// around the removed node.
//
// TODO: this method only makes sense in the context of a PrefixSet.
// "subtracting" a whole key-value entry from another isn't meaningful. So
// maybe we need two types of trees: value-bearing ones, and others that just
// have value-less entries.
func (t *tree[T]) subtractTree(o tree[T]) *tree[T] {
	if o.hasEntry {
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

func (t *tree[T]) isEmpty() bool {
	return t.key.isZero() && t.left == nil && t.right == nil
}

// Helpful for debugging
var debugDepth = 0

func debugf(s0 string, rest ...any) {
	//go:coverage ignore
	indentStr := strings.Repeat("  ", debugDepth)
	fmt.Printf(indentStr+s0, rest...)
}

// union modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
// TODO: lots of duplicated code here
func (t *tree[T]) unionTree(o tree[T]) *tree[T] {
	// if o is empty, then the union is just t
	if o.isEmpty() {
		return t
	}

	if t.key.equalFromRoot(o.key) {
		if !t.hasEntry {
			t.setValueFrom(&o)
		}

		for _, direction := range []bool{false, true} {
			tChild, oChild := t.child(direction), o.child(direction)
			if *oChild != nil {
				tNext := &t
				if *tChild != nil {
					tNext = tChild
				}
				*tNext = (*tNext).unionTree(**oChild)
			}
		}
		return t
	}

	common := t.key.commonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	case common == t.key.len:
		// Traverse t in the direction of o
		tChildFollow := t.childPtr(o.key, common)
		if *tChildFollow == nil {
			*tChildFollow = o.copy()
			(*tChildFollow).key.offset = t.key.len
		} else {
			*tChildFollow = (*tChildFollow).unionTree(o)
		}
	// o.key is a prefix of t.key
	case common == o.key.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry; if the node exists in the o tree, it will need to be in the
		// union tree. Insert it and continue traversing from there.
		newNode := newTree[T](o.key).setValueFrom(&o).setChildFromBit(t.key, o.key.len, t)
		t.key.offset = newNode.key.len
		t = newNode.unionTree(o)
	}
	return t
}

func (t *tree[T]) intersectTreeImpl(
	o tree[T],
	tPathHasEntry, oPathHasEntry bool,
) *tree[T] {

	// if o is an empty tree, then any intersection with it is also empty
	if o.isEmpty() {
		return &tree[T]{}
	}

	if t.key.equalFromRoot(o.key) {
		// Consider t and o themselves.
		//
		// If there is no entry in o at t.key or above it, then remove t's entry.
		//
		// TODO should this be t.remove(t.key)? Could we end up with an
		// unnecessary prefix node?
		if t.hasEntry && !(o.hasEntry || oPathHasEntry) {
			t.clearValue()
			// We need to remember that t had an entry here so that o's
			// descendants are kept
			tPathHasEntry = true
		}

		// Consider the children of t and o
		for _, direction := range []bool{false, true} {
			tChild, oChild := t.child(direction), o.child(direction)
			switch {
			case *tChild == nil && *oChild != nil && (t.hasEntry || tPathHasEntry):
				*tChild = (*oChild).copy()
			case *tChild != nil && *oChild == nil && !(o.hasEntry || oPathHasEntry):
				*tChild = nil
			case *tChild != nil && *oChild != nil:
				(*tChild).intersectTreeImpl(
					**oChild,
					t.hasEntry || tPathHasEntry,
					o.hasEntry || oPathHasEntry,
				)
			}
		}
		return t
	}

	common := t.key.commonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	// e.g. t=00, o=000
	case common == t.key.len:
		// If t is an entry then we keep everything under both t and o, since t
		// is an ancestor entry to both.
		if t.hasEntry {
			if !oPathHasEntry {
				t.clearValue()
			}
			t = t.unionTree(o)
		} else {
			// t forks in the middle of o.key. To take the intersection, we
			// need to traverse t toward o.key and prune the other child of t.
			//
			// The bit of o.key just after the common prefix determines which
			// of t's children to follow and which to remove.
			// e.g. t=00, o=000 -> follow left, remove right
			tChildFollow, tChildRemove := t.childPtrs(o.key, common)

			// Traverse t in the direction of o.key.
			if *tChildFollow != nil {
				(*tChildFollow).intersectTreeImpl(o,
					t.hasEntry || tPathHasEntry,
					o.hasEntry || oPathHasEntry,
				)
			}

			// Remove the child of t that diverges from o.
			//
			// Exception: if o has an ancestor entry, then we don't need to remove
			// anything under t. TODO: is this check necessary?
			if !oPathHasEntry {
				*tChildRemove = nil
			}
		}

	// o.key is a prefix of t.key
	// e.g. t=000, o=00
	case common == o.key.len:
		// o forks in the middle of t.key. Similar to above.
		oChildFollow := o.childPtr(t.key, common)

		// Traverse o in the direction of t.key.
		//
		// We don't need to visit t's children here; if there is intersection
		// under t, it will be handled within the call below by one of the
		// above cases.
		if *oChildFollow != nil {
			t.intersectTreeImpl(**oChildFollow,
				t.hasEntry || tPathHasEntry,
				o.hasEntry || oPathHasEntry,
			)
		}
	}

	return t
}

// intersectTree modifies t so that it is the intersection of the entries of t
// and o: an entry is included iff it (1) is present in both trees or (2) is
// present in one tree and has a parent entry in the other tree.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
func (t *tree[T]) intersectTree(o tree[T]) *tree[T] {
	return t.intersectTreeImpl(o, false, false)
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
			if n.key.equalFromRoot(k) && n.hasEntry {
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
		if ret = (n.key.equalFromRoot(k) && n.hasEntry); ret {
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
		ret = n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasEntry
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
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasEntry {
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
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasEntry {
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
		if n.hasEntry {
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
		if n.hasEntry && o.encompasses(n.key, false) {
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
		if !n.hasEntry {
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
