package netipds

import (
	"fmt"
)

// tree is a binary radix tree supporting 128-bit keys (see key.go).
//
// The tree is compressed by default, however it supports uncompressed
// insertion via insertLazy(). This can be much faster than insert() and works
// well with netipds's intended usage pattern (build a collection with a
// builder type, then generate an immutable version). After lazy insertions,
// the tree can be compressed using the compress() method.
type tree[T any, K Key[K]] struct {
	key      K
	hasEntry bool
	value    T
	left     *tree[T, K]
	right    *tree[T, K]
}

// newTree returns a new tree with the provided key.
func newTree[T any, K Key[K]](k K) *tree[T, K] {
	return &tree[T, K]{key: k}
}

// setValue sets t's value to v and returns t.
func (t *tree[T, K]) setValue(v T) *tree[T, K] {
	t.value = v
	t.hasEntry = true
	return t
}

// clearValue removes the value from t.
func (t *tree[T, K]) clearValue() {
	var zeroVal T
	t.value = zeroVal
	t.hasEntry = false
}

// setValueFrom sets t's value to o's value and returns t.
func (t *tree[T, K]) setValueFrom(o *tree[T, K]) *tree[T, K] {
	if o.hasEntry {
		return t.setValue(o.value)
	}
	return t
}

// child returns a pointer to the specified child of t.
func (t *tree[T, K]) child(b bit) **tree[T, K] {
	if b == bitR {
		return &t.right
	}
	return &t.left
}

// children returns pointers to t's children.
func (t *tree[T, K]) children(whichFirst bit) (a **tree[T, K], b **tree[T, K]) {
	if whichFirst == bitR {
		return &t.right, &t.left
	}
	return &t.left, &t.right
}

// setChild sets one of t's children to n, if it isn't already set, choosing
// which child based on the bit at n.key.Offset(). A provided nil is ignored.
func (t *tree[T, K]) setChild(n *tree[T, K]) *tree[T, K] {
	child := t.child(n.key.Bit(n.key.Offset()))
	if *child == nil && n != nil {
		*child = n
	}
	return t
}

// copy returns a copy of t, creating copies of all of t's descendants in the
// process.
func (t *tree[T, K]) copy() *tree[T, K] {
	ret := newTree[T](t.key)
	if t.left != nil {
		ret.left = t.left.copy()
	}
	if t.right != nil {
		ret.right = t.right.copy()
	}
	ret.setValueFrom(t)
	return ret
}

func (t *tree[T, K]) stringImpl(indent string, pre string, hideVal bool) string {
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

func (t *tree[T, K]) String() string {
	return t.stringImpl("", "", false)
}

// size returns the number of nodes within t that have values.
// TODO: keep track of this instead of calculating it lazily
func (t *tree[T, K]) size() int {
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
func (t *tree[T, K]) insert(k K, v T) *tree[T, K] {
	// Inserting at t itself
	if t.key.EqualFromRoot(k) {
		return t.setValue(v)
	}

	common := t.key.CommonPrefixLen(k)
	switch {
	// Inserting at a descendant; recurse into the appropriate child
	case common == t.key.Len():
		child := t.child(k.Bit(t.key.Len()))
		if *child == nil {
			*child = newTree[T](k.Rest(t.key.Len())).setValue(v)
		}
		*child = (*child).insert(k, v)
		return t
	// Inserting at a prefix of t.key; create a new parent node with t as its
	// sole child
	case common == k.Len():
		return t.newParent(k).setValue(v)
	// Neither is a prefix of the other; create a new parent at their common
	// prefix with children t and its new sibling
	default:
		return t.newParent(t.key.Truncated(common)).setChild(
			newTree[T](k.Rest(common)).setValue(v),
		)
	}
}

// insertLazy inserts value v at key k without path compression.
func (t *tree[T, K]) insertLazy(k K, v T) *tree[T, K] {
	switch {
	// Inserting at t itself
	case t.key.EqualFromRoot(k):
		return t.setValue(v)
	// Inserting at a descendant
	case t.key.CommonPrefixLen(k) == t.key.Len():
		bit := k.Bit(t.key.Len())
		child := t.child(bit)
		if *child == nil {
			*child = newTree[T](t.key.Next(bit))
		}
		(*child).insertLazy(k, v)
		return t
	// Nothing to do
	default:
		return t
	}
}

// compress performs path compression on tree t.
func (t *tree[T, K]) compress() *tree[T, K] {
	switch {
	case t.left == nil && t.right == nil:
		return t
	case t.left == nil:
		t.right.key.SetOffset(t.key.Offset())
		return t.right
	case t.right == nil:
		t.left.key.SetOffset(t.key.Offset())
		return t.left
	default:
		return t
	}
}

// remove removes the exact provided key from the tree, if it exists, and
// performs path compression.
func (t *tree[T, K]) remove(k K) *tree[T, K] {
	switch {
	// Removing t itself
	case k.EqualFromRoot(t.key):
		if t.hasEntry {
			t.clearValue()
		}
		switch {
		// No children (deleting a leaf node)
		case t.left == nil && t.right == nil:
			return nil
		// Only one child; merge with it
		case t.left == nil:
			t.right.key.SetOffset(t.key.Offset())
			return t.right
		case t.right == nil:
			t.left.key.SetOffset(t.key.Offset())
			return t.left
		// t is a shared prefix node, so it can't be removed
		default:
			return t
		}
	// Removing a descendant of t; recurse into the appropriate child
	case t.key.IsPrefixOf(k, false):
		child := t.child(k.Bit(t.key.Len()))
		if *child != nil {
			*child = (*child).remove(k)
		}
		return t
	// Nothing to do
	default:
		return t
	}
}

// subtractKey removes k and all of its descendants from the tree, leaving the
// remaining key space behind. If k is a descendant of t, then new nodes may be
// created to fill in the gaps around k.
func (t *tree[T, K]) subtractKey(k K) *tree[T, K] {
	// This whole branch is being subtracted; no need to traverse further
	if t.key.EqualFromRoot(k) || k.IsPrefixOf(t.key, false) {
		return nil
	}
	// A child of t is being subtracted
	if t.key.IsPrefixOf(k, false) {
		child := t.child(k.Bit(t.key.Len()))
		if *child != nil {
			*child = (*child).subtractKey(k.Rest(t.key.Len()))
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
func (t *tree[T, K]) subtractTree(o *tree[T, K]) *tree[T, K] {
	if o.hasEntry {
		// This whole branch is being subtracted; no need to traverse further
		if o.key.IsPrefixOf(t.key, false) {
			return nil
		}
		// A descendant of t is being subtracted
		if t.key.IsPrefixOf(o.key, false) {
			t.insertHole(o.key, t.value)
		}
	}
	// Consider the children of both t and o
	for _, bit := range eachBit {
		tChild, oChild := t.child(bit), o.child(bit)
		if *oChild != nil {
			if *tChild == nil {
				tChild = &t
			}
			*tChild = (*tChild).subtractTree(*oChild)
		}
	}
	return t
}

func (t *tree[T, K]) isEmpty() bool {
	return t.key.IsZero() && t.left == nil && t.right == nil
}

// newParent returns a new node with key k whose sole child is t.
func (t *tree[T, K]) newParent(k K) *tree[T, K] {
	t.key.SetOffset(k.Len())
	parent := newTree[T](k).setChild(t)
	return parent
}

// mergeTree modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
// TODO: lots of duplicated code here
func (t *tree[T, K]) mergeTree(o *tree[T, K]) *tree[T, K] {
	// If o is empty, then the union is just t
	if o.isEmpty() {
		return t
	}

	if t.key.EqualFromRoot(o.key) {
		if !t.hasEntry {
			t.setValueFrom(o)
		}

		for _, bit := range eachBit {
			tChild, oChild := t.child(bit), o.child(bit)
			if *oChild != nil {
				tNext := &t
				if *tChild != nil {
					tNext = tChild
				}
				*tNext = (*tNext).mergeTree(*oChild)
			}
		}
		return t
	}

	common := t.key.CommonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	case common == t.key.Len():
		// Traverse t in the direction of o
		tChildFollow := t.child(o.key.Bit(t.key.Len()))
		if *tChildFollow == nil {
			*tChildFollow = o.copy()
			(*tChildFollow).key.SetOffset(t.key.Len())
		} else {
			*tChildFollow = (*tChildFollow).mergeTree(o)
		}
		return t
	// o.key is a prefix of t.key
	case common == o.key.Len():
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.newParent(o.key).setValueFrom(o).mergeTree(o)
	// Neither is a prefix of the other
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value.
		return t.newParent(t.key.Truncated(common)).setChild(
			newTree[T](o.key.Rest(common)).setValueFrom(o),
		)
	}
}

func (t *tree[T, K]) intersectTreeImpl(
	o *tree[T, K],
	tPathHasEntry, oPathHasEntry bool,
) *tree[T, K] {

	// If o is an empty tree, then any intersection with it is also empty
	if o.isEmpty() {
		return &tree[T, K]{}
	}

	if t.key.EqualFromRoot(o.key) {
		// Consider t and o themselves.
		// If there is no entry in o at t.key or above it, then remove t's
		// entry.
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
		for _, bit := range eachBit {
			tChild, oChild := t.child(bit), o.child(bit)
			switch {
			case *tChild == nil && *oChild != nil && (t.hasEntry || tPathHasEntry):
				*tChild = (*oChild).copy()
			case *tChild != nil && *oChild == nil && !(o.hasEntry || oPathHasEntry):
				*tChild = nil
			case *tChild != nil && *oChild != nil:
				*tChild = (*tChild).intersectTreeImpl(
					*oChild,
					t.hasEntry || tPathHasEntry,
					o.hasEntry || oPathHasEntry,
				)
			}
		}
		return t
	}

	common := t.key.CommonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	case common == t.key.Len():
		if t.hasEntry {
			if !oPathHasEntry {
				t.clearValue()
			}
			t = t.insert(o.key, o.value)
		}

		// t forks in the middle of o.key. To take the intersection, we
		// need to traverse t toward o.key and prune the other child of t.
		//
		// The bit of o.key just after the common prefix determines which
		// of t's children to follow and which to remove.
		// e.g. t=00, o=000 -> follow left, remove right
		tChildFollow, tChildRemove := t.children(o.key.Bit(common))

		// Traverse t in the direction of o.key.
		if *tChildFollow != nil {
			*tChildFollow = (*tChildFollow).intersectTreeImpl(o,
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

	// o.key is a prefix of t.key
	case common == o.key.Len():
		// o forks in the middle of t.key. Similar to above.
		oChildFollow := o.child(t.key.Bit(common))

		// Traverse o in the direction of t.key.
		//
		// We don't need to visit t's children here; if there is intersection
		// under t, it will be handled within the call below by one of the
		// above cases.
		if *oChildFollow != nil {
			t = t.intersectTreeImpl(*oChildFollow,
				t.hasEntry || tPathHasEntry,
				o.hasEntry || oPathHasEntry,
			)
		}
	// Neither is a prefix of the other, so the intersection is empty
	default:
		t = nil
	}

	return t
}

// intersectTree modifies t so that it is the intersection of the entries of t
// and o: an entry is included iff it (1) is present in both trees or (2) is
// present in one tree and has a parent entry in the other tree.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
func (t *tree[T, K]) intersectTree(o *tree[T, K]) *tree[T, K] {
	return t.intersectTreeImpl(o, false, false)
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *tree[T, K]) insertHole(k K, v T) *tree[T, K] {
	switch {
	// Removing t itself (no descendants will receive v)
	case t.key.EqualFromRoot(k):
		return nil
	// k is a descendant of t; start digging a hole to k
	case t.key.IsPrefixOf(k, false):
		t.clearValue()
		// Create a new sibling to receive v if needed, then continue traversing
		bit := k.Bit(t.key.Len())
		child, sibling := t.children(bit)
		if *sibling == nil {
			*sibling = newTree[T](t.key.Next((^bit) & 1)).setValue(v)
		}
		*child = newTree[T](t.key.Next(bit)).insertHole(k, v)
		return t
	// Nothing to do
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
func (t *tree[T, K]) walk(path K, fn func(*tree[T, K]) bool) {
	// Follow provided path directly until it's exhausted
	n := t
	for n != nil && n.key.Len() < path.Len() {
		if !n.key.IsZero() {
			if fn(n) {
				return
			}
		}
		n = *(n.child(path.Bit(n.key.CommonPrefixLen(path))))
	}

	if n == nil {
		return
	}

	// After path is exhausted, visit all children
	var st stack[*tree[T, K]]
	var stop bool
	st.Push(n)
	for !st.IsEmpty() {
		stop = false
		if n = st.Pop(); n == nil {
			continue
		}
		if !n.key.IsZero() {
			stop = fn(n)
		}
		if n.key.Len() < 128 && !stop {
			st.Push(n.right)
			st.Push(n.left)
		}
	}
}

// pathNext returns the child of t which is next in the traversal of the
// specified path.
func (t *tree[T, K]) pathNext(path K) *tree[T, K] {
	if path.Bit(t.key.Len()) == bitR {
		return t.right
	}
	return t.left
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T, K]) get(k K) (val T, ok bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if n.key.Len() >= k.Len() {
			if n.key.EqualFromRoot(k) && n.hasEntry {
				val, ok = n.value, true
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *tree[T, K]) contains(k K) (ret bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if ret = n.key.EqualFromRoot(k) && n.hasEntry; ret {
			break
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
func (t *tree[T, K]) encompasses(k K, strict bool) (ret bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if ret = n.hasEntry && n.key.IsPrefixOf(k, strict); ret {
			break
		}
	}
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T, K]) rootOf(k K, strict bool) (outKey K, val T, ok bool) {
	t.walk(k, func(n *tree[T, K]) bool {
		if n.hasEntry && n.key.IsPrefixOf(k, strict) {
			outKey, val, ok = n.key, n.value, true
			return true
		}
		return false
	})
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T, K]) parentOf(k K, strict bool) (outKey K, val T, ok bool) {
	t.walk(k, func(n *tree[T, K]) bool {
		if n.hasEntry && n.key.IsPrefixOf(k, strict) {
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
func (t *tree[T, K]) descendantsOf(k K, strict bool) (ret *tree[T, K]) {
	ret = &tree[T, K]{}
	t.walk(k, func(n *tree[T, K]) bool {
		if k.IsPrefixOf(n.key, false) {
			ret.key = n.key.Rooted()
			ret.left = n.left
			ret.right = n.right
			if !(strict && n.key.EqualFromRoot(k)) {
				ret.setValueFrom(n)
			}
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
func (t *tree[T, K]) ancestorsOf(k K, strict bool) (ret *tree[T, K]) {
	ret = &tree[T, K]{}
	t.walk(k, func(n *tree[T, K]) bool {
		if !n.key.IsPrefixOf(k, false) {
			return true
		}
		if n.hasEntry && !(strict && n.key.EqualFromRoot(k)) {
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
func (t *tree[T, K]) filter(o *tree[bool, K]) {
	remove := make([]K, 0)
	var k K
	t.walk(k, func(n *tree[T, K]) bool {
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
// TODO: does it make sense to have both this method and filter()?
func (t *tree[T, K]) filterCopy(o *tree[bool, K]) *tree[T, K] {
	ret := &tree[T, K]{}
	var k K
	t.walk(k, func(n *tree[T, K]) bool {
		if n.hasEntry && o.encompasses(n.key, false) {
			ret = ret.insert(n.key, n.value)
		}
		return false
	})
	return ret
}

// overlapsKey reports whether any key in t overlaps k.
func (t *tree[T, K]) overlapsKey(k K) bool {
	var ret bool
	t.walk(k, func(n *tree[T, K]) bool {
		if !n.hasEntry {
			return false
		}
		if n.key.IsPrefixOf(k, false) || k.IsPrefixOf(n.key, false) {
			ret = true
			return true
		}
		return false
	})
	return ret
}
