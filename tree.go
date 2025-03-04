package netipds

import (
	"fmt"
)

// tree is a binary radix tree.
type tree[T any, B keyBits[B]] struct {
	key      key[B]
	hasEntry bool
	value    T
	left     *tree[T, B]
	right    *tree[T, B]
}

// newTree returns a new tree with the provided key.
func newTree[T any, B keyBits[B]](k key[B]) *tree[T, B] {
	return &tree[T, B]{key: k}
}

// setValue sets t's value to v and returns t.
func (t *tree[T, B]) setValue(v T) *tree[T, B] {
	t.value = v
	t.hasEntry = true
	return t
}

// clearValue removes the value from t.
func (t *tree[T, B]) clearValue() {
	var zeroVal T
	t.value = zeroVal
	t.hasEntry = false
}

// setValueFrom sets t's value to o's value and returns t.
func (t *tree[T, B]) setValueFrom(o *tree[T, B]) *tree[T, B] {
	if o.hasEntry {
		return t.setValue(o.value)
	}
	return t
}

// child returns a pointer to the specified child of t.
func (t *tree[T, B]) child(b bit) **tree[T, B] {
	if b == bitR {
		return &t.right
	}
	return &t.left
}

// children returns pointers to t's children.
func (t *tree[T, B]) children(whichFirst bit) (a **tree[T, B], b **tree[T, B]) {
	if whichFirst == bitR {
		return &t.right, &t.left
	}
	return &t.left, &t.right
}

// setChild sets one of t's children to n, if it isn't already set, choosing
// which child based on the bit at n.key.offset. A provided nil is ignored.
func (t *tree[T, B]) setChild(n *tree[T, B]) *tree[T, B] {
	child := t.child(n.key.Bit(n.key.offset))
	if *child == nil && n != nil {
		*child = n
	}
	return t
}

// copy returns a copy of t, creating copies of all of t's descendants in the
// process.
func (t *tree[T, B]) copy() *tree[T, B] {
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

func (t *tree[T, B]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	if hideVal {
		ret = fmt.Sprintf("%s%s%s\n", indent, pre, t.key.String())
	} else {
		ret = fmt.Sprintf("%s%s%s: %v\n", indent, pre, t.key.String(), t.value)
	}
	if t.left != nil {
		ret += t.left.stringImpl(indent+"  ", "L:", hideVal)
	}
	if t.right != nil {
		ret += t.right.stringImpl(indent+"  ", "R:", hideVal)
	}
	return ret
}

func (t *tree[T, B]) String() string {
	return t.stringImpl("", "", false)
}

// size returns the number of nodes within t that have values.
// TODO: keep track of this instead of calculating it lazily
func (t *tree[T, B]) size() int {
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
func (t *tree[T, B]) insert(k key[B], v T) *tree[T, B] {
	// Inserting at t itself
	if t.key.EqualFromRoot(k) {
		return t.setValue(v)
	}

	common := t.key.CommonPrefixLen(k)
	switch {
	// Inserting at a descendant; recurse into the appropriate child
	case common == t.key.len:
		child := t.child(k.Bit(t.key.len))
		if *child == nil {
			*child = newTree[T](k.Rest(t.key.len)).setValue(v)
		}
		*child = (*child).insert(k, v)
		return t
	// Inserting at a prefix of t.key; create a new parent node with t as its
	// sole child
	case common == k.len:
		return t.newParent(k).setValue(v)
	// Neither is a prefix of the other; create a new parent at their common
	// prefix with children t and its new sibling
	default:
		return t.newParent(t.key.Truncated(common)).setChild(
			newTree[T](k.Rest(common)).setValue(v),
		)
	}
}

// remove removes the exact provided key from the tree, if it exists, and
// performs path compression.
func (t *tree[T, B]) remove(k key[B]) *tree[T, B] {
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
			t.right.key.offset = t.key.offset
			return t.right
		case t.right == nil:
			t.left.key.offset = t.key.offset
			return t.left
		// t is a shared prefix node, so it can't be removed
		default:
			return t
		}
	// Removing a descendant of t; recurse into the appropriate child
	case t.key.IsPrefixOf(k):
		child := t.child(k.Bit(t.key.len))
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
func (t *tree[T, B]) subtractKey(k key[B]) *tree[T, B] {
	// Subtracting from empty tree yields empty tree
	if t.isEmpty() {
		return t
	}
	// This tree is a child of what's being subtracted; no need to traverse further
	if t.key.EqualFromRoot(k) || k.IsPrefixOf(t.key) {
		return nil
	}
	// A child of t is being subtracted
	if t.key.IsPrefixOf(k) {
		child := t.child(k.Bit(t.key.len))
		if *child != nil {
			*child = (*child).subtractKey(k.Rest(t.key.len))
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
func (t *tree[T, B]) subtractTree(o *tree[T, B]) *tree[T, B] {
	// Subtracting from empty tree yields empty tree
	if t.isEmpty() {
		return t
	}
	if o.hasEntry {
		// This tree is a child of what's being subtracted; no need to traverse further
		if o.key.IsPrefixOf(t.key) {
			return nil
		}
		// A descendant of t is being subtracted
		if t.key.IsPrefixOf(o.key) {
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

func (t *tree[T, B]) isEmpty() bool {
	return t.key.IsZero() && t.left == nil && t.right == nil
}

// newParent returns a new node with key k whose sole child is t.
func (t *tree[T, B]) newParent(k key[B]) *tree[T, B] {
	t.key.offset = (k.len)
	parent := newTree[T](k).setChild(t)
	return parent
}

// mergeTree modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
// TODO: lots of duplicated code here
func (t *tree[T, B]) mergeTree(o *tree[T, B]) *tree[T, B] {
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
	case common == t.key.len:
		// Traverse t in the direction of o
		tChildFollow := t.child(o.key.Bit(t.key.len))
		if *tChildFollow == nil {
			*tChildFollow = o.copy()
			(*tChildFollow).key.offset = t.key.len
		} else {
			*tChildFollow = (*tChildFollow).mergeTree(o)
		}
		return t
	// o.key is a prefix of t.key
	case common == o.key.len:
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

func (t *tree[T, B]) intersectTreeImpl(
	o *tree[T, B],
	tPathHasEntry, oPathHasEntry bool,
) *tree[T, B] {

	// If o is an empty tree, then any intersection with it is also empty
	if o.isEmpty() {
		return &tree[T, B]{}
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
	case common == t.key.len:
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
	case common == o.key.len:
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
func (t *tree[T, B]) intersectTree(o *tree[T, B]) *tree[T, B] {
	return t.intersectTreeImpl(o, false, false)
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *tree[T, B]) insertHole(k key[B], v T) *tree[T, B] {
	switch {
	// Removing t itself (no descendants will receive v)
	case t.key.EqualFromRoot(k):
		return nil
	// k is a descendant of t; start digging a hole to k
	case t.key.IsPrefixOf(k):
		t.clearValue()
		// Create a new sibling to receive v if needed, then continue traversing
		bit := k.Bit(t.key.len)
		child, sibling := t.children(bit)
		if *sibling == nil {
			*sibling = newTree[T](t.key.Next(!bit)).setValue(v)
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
func (t *tree[T, B]) walk(path key[B], fn func(*tree[T, B]) bool) {
	// Follow provided path directly until it's exhausted
	n := t
	for n != nil && n.key.len < path.len {
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
	var st stack[*tree[T, B]]
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
		if n.key.len < 128 && !stop {
			st.Push(n.right)
			st.Push(n.left)
		}
	}
}

// pathNext returns the child of t which is next in the traversal of the
// specified path.
func (t *tree[T, B]) pathNext(path uint128) *tree[T, B] {
	if path.Bit(t.key.len) {
		return t.right
	}
	return t.left
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T, B]) get(k key[B]) (val T, ok bool) {
	u128 := k.content.U128()
	for n := t.pathNext(u128); n != nil; n = n.pathNext(u128) {
		if n.key.len >= k.len {
			if n.key.EqualFromRoot(k) && n.hasEntry {
				val, ok = n.value, true
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *tree[T, B]) contains(k key[B]) (ret bool) {
	u128 := k.content.U128()
	for n := t.pathNext(u128); n != nil; n = n.pathNext(u128) {
		if ret = n.key.EqualFromRoot(k) && n.hasEntry; ret {
			break
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses or is equal to the provided key.
func (t *tree[T, B]) encompasses(k key[B]) (ret bool) {
	u128 := k.content.U128()
	for n := t.pathNext(u128); n != nil; n = n.pathNext(u128) {
		if ret = n.hasEntry && n.key.IsPrefixOf(k); ret {
			break
		}
	}
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
func (t *tree[T, B]) rootOf(k key[B]) (outKey key[B], val T, ok bool) {
	u128 := k.content.U128()
	for n := t.pathNext(u128); n != nil; n = n.pathNext(u128) {
		if ok = n.hasEntry && n.key.IsPrefixOf(k); ok {
			return n.key, n.value, ok
		}
	}
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
func (t *tree[T, B]) parentOf(k key[B]) (outKey key[B], val T, ok bool) {
	u128 := k.content.U128()
	for n := t.pathNext(u128); n != nil; n = n.pathNext(u128) {
		if n.hasEntry && n.key.IsPrefixOf(k) {
			outKey, val, ok = n.key, n.value, true
		}
	}
	return
}

// descendantsOf returns the sub-tree containing all descendants of the
// provided key. The key itself will be included if it has an entry in the
// tree. descendantsOf returns an empty tree if the provided key is not in the
// tree.
func (t *tree[T, B]) descendantsOf(k key[B]) (ret *tree[T, B]) {
	ret = &tree[T, B]{}
	t.walk(k, func(n *tree[T, B]) bool {
		if k.IsPrefixOf(n.key) {
			ret.key = n.key.Rooted()
			ret.left = n.left
			ret.right = n.right
			ret.setValueFrom(n)
			return true
		}
		return false
	})
	return
}

// ancestorsOf returns the sub-tree containing all ancestors of the provided
// key. The key itself will be included if it has an entry in the tree.
// ancestorsOf returns an empty tree if key has no ancestors in the tree.
func (t *tree[T, B]) ancestorsOf(k key[B]) (ret *tree[T, B]) {
	ret = &tree[T, B]{}
	t.walk(k, func(n *tree[T, B]) bool {
		if !n.key.IsPrefixOf(k) {
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
func (t *tree[T, B]) filter(o *tree[bool, B]) {
	remove := make([]key[B], 0)
	var k key[B]
	t.walk(k, func(n *tree[T, B]) bool {
		if !o.encompasses(n.key) {
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
func (t *tree[T, B]) filterCopy(o *tree[bool, B]) *tree[T, B] {
	ret := &tree[T, B]{}
	var k key[B]
	t.walk(k, func(n *tree[T, B]) bool {
		if n.hasEntry && o.encompasses(n.key) {
			ret = ret.insert(n.key, n.value)
		}
		return false
	})
	return ret
}

// overlapsKey reports whether any key in t overlaps k.
func (t *tree[T, B]) overlapsKey(k key[B]) bool {
	var ret bool
	t.walk(k, func(n *tree[T, B]) bool {
		if !n.hasEntry {
			return false
		}
		if n.key.IsPrefixOf(k) || k.IsPrefixOf(n.key) {
			ret = true
			return true
		}
		return false
	})
	return ret
}
