package netipds

import (
	"fmt"
)

// tree is a binary radix tree supporting 128-bit keys.
//
// The tree is partitioned depth-wise into halves: each key's offset and len
// must both be in the range [0, 63] or [64, 127].
//
// The tree is compressed by default, however it supports uncompressed
// insertion via insertLazy(). This can be much faster than insert() and works
// well with netipds's intended usage pattern (build a collection with a
// builder type, then generate an immutable version). After lazy insertions,
// the tree can be compressed using the compress() method.
type tree[T any] struct {
	halfkey  halfkey
	hasEntry bool
	value    T
	left     *tree[T]
	right    *tree[T]
}

// newTree returns a new tree node with the provided halfkey.
func newTree[T any](h halfkey) *tree[T] {
	return &tree[T]{halfkey: h}
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
	if o.hasEntry {
		return t.setValue(o.value)
	}
	return t
}

// child returns a pointer to the child of t selected by b.
func (t *tree[T]) child(b bit) **tree[T] {
	if b == bitR {
		return &t.right
	}
	return &t.left
}

// children returns pointers to t's children.
func (t *tree[T]) children(whichFirst bit) (a **tree[T], b **tree[T]) {
	if whichFirst == bitR {
		return &t.right, &t.left
	}
	return &t.left, &t.right
}

// setChild sets one of t's children to n, if it isn't already set, choosing
// which child based on the bit at n.halfkey.offset. A provided nil is ignored.
func (t *tree[T]) setChild(n *tree[T]) *tree[T] {
	child := t.child(n.halfkey.bit(n.halfkey.offset))
	if *child == nil && n != nil {
		*child = n
	}
	return t
}

// copy returns a copy of t, creating copies of all of t's descendants in the
// process.
func (t *tree[T]) copy() *tree[T] {
	ret := newTree[T](t.halfkey)
	if t.left != nil {
		ret.left = t.left.copy()
	}
	if t.right != nil {
		ret.right = t.right.copy()
	}
	ret.setValueFrom(t)
	return ret
}

func (t *tree[T]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	if hideVal {
		ret = fmt.Sprintf("%s%s%s\n", indent, pre, t.halfkey.StringRel())
	} else {
		ret = fmt.Sprintf("%s%s%s: %v\n", indent, pre, t.halfkey.StringRel(), t.value)
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

// size returns the number of entries in t.
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
	// Inserting at t itself
	if t.halfkey.keyEqualFromRoot(k) {
		return t.setValue(v)
	}

	// TODO what if t is hi and k is low?
	// Seems like we might need insertKey and insertHalfkey?
	// Where insertKey is the entrypoint and insertHalfkey is inner
	common := t.halfkey.keyCommonPrefixLen(k)
	switch {
	// Inserting at a descendant; recurse into the appropriate child.
	//
	// Note: in this case, k is strictly longer than t.halfkey (k.len > t.halfkey.len),
	// because (1) t.halfkey is itself the common prefix and (2) the check above
	// ruled out the possibility that k == t.halfkey.
	case common == t.halfkey.len:
		// t.halfkey: 000
		// k:     0001
		//           ^ this bit determines which child of t to select
		child := t.child(k.bit(t.halfkey.len))
		if *child == nil {
			var kRestHalf halfkey
			kRest := k.rest(t.halfkey.len)
			switch {
			// t.halfkey and k are both contained in hi partition
			// . t.halfkey: 000
			// . k:     0000
			case t.halfkey.len < 64 && k.len <= 64:
				kRestHalf = halfkey{kRest.content.hi, kRest.offset, kRest.len}
			// t.halfkey ends in hi partition; k ends in lo partition
			// . t.halfkey: 000
			// . k:     0000 01
			// We need to create a transition node to fill out hi and be a parent of k
			// TODO: does it matter in which partition k starts?
			// - if k starts in lo, this is fine
			// - if k starts in hi, we still need to bridge the gap, right?
			case t.halfkey.len < 64 && k.len > 64:
				*child = newTree[T](halfkey{kRest.content.hi, kRest.offset, 64})
				kRestHalf = halfkey{kRest.content.lo, 64, kRest.len}
			// t and k both end in the lo partition
			// . t.halfkey: 0000 0
			// . k:     0000 01
			// TODO what if k _starts_ in lo partition? I guess that shouldn't
			// happen
			case t.halfkey.len >= 64 && k.len > 64:
				kRestHalf = halfkey{kRest.content.lo, kRest.offset, kRest.len}
			default:
				// all other cases impossible (k is strictly longer than t.halfkey)
				panic("unreachable")
			}
			*child = newTree[T](kRestHalf).setValue(v)
			// TODO this is an optimization that wasn't here before the 64-bit
			// key refactor
			return t
		}
		*child = (*child).insert(k, v)
		return t
	// Inserting at a prefix of t.halfkey; create a new parent node with t as its
	// sole child.
	//
	// Note: in this case, t.halfkey is strictly longer than k (t.halfkey.len > k.len),
	// because (1) k is itself the common prefix and (2) the check above ruled
	// out the possibility that k == t.halfkey.
	case common == k.len:
		// We also know that t.halfkey and k end in the same partition. All
		// insertions into the hi partition must hit at least one hi-partition
		// node (a node can't start at root and end in the lo partition).

		if (t.halfkey.len > 64) != (k.len > 64) {
			panic("unreachable")
		}
		// Need a function on key that "gets the halfkey you want"
		return t.newParent(k).setValue(v)
	// Neither is a prefix of the other; create a new parent at their common
	// prefix, with children t and its new sibling.
	default:
		return t.newParent(t.halfkey.truncated(common)).setChild(
			newTree[T](k.rest(common)).setValue(v),
		)
	}
}

// insertLazy inserts value v at key k without path compression.
func (t *tree[T]) insertLazy(k halfkey, v T) *tree[T] {
	switch {
	// Inserting at t itself
	case t.halfkey.equalFromRoot(k):
		return t.setValue(v)
	// Inserting at a descendant
	case t.halfkey.commonPrefixLen(k) == t.halfkey.len:
		bit := k.bit(t.halfkey.len)
		child := t.child(bit)
		if *child == nil {
			*child = newTree[T](t.halfkey.next(bit))
		}
		(*child).insertLazy(k, v)
		return t
	// Nothing to do
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
		t.right.halfkey.offset = t.halfkey.offset
		return t.right
	case t.right == nil:
		t.left.halfkey.offset = t.halfkey.offset
		return t.left
	default:
		return t
	}
}

// remove removes the exact provided key from the tree, if it exists, and
// performs path compression.
func (t *tree[T]) remove(k halfkey) *tree[T] {
	switch {
	// Removing t itself
	case k.equalFromRoot(t.halfkey):
		if t.hasEntry {
			t.clearValue()
		}
		switch {
		// No children (deleting a leaf node)
		case t.left == nil && t.right == nil:
			return nil
		// Only one child; merge with it
		case t.left == nil:
			t.right.halfkey.offset = t.halfkey.offset
			return t.right
		case t.right == nil:
			t.left.halfkey.offset = t.halfkey.offset
			return t.left
		// t is a shared prefix node, so it can't be removed
		default:
			return t
		}
	// Removing a descendant of t; recurse into the appropriate child
	case t.halfkey.isPrefixOf(k, false):
		child := t.child(k.bit(t.halfkey.len))
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
func (t *tree[T]) subtractKey(k halfkey) *tree[T] {
	// This whole branch is being subtracted; no need to traverse further
	if t.halfkey.equalFromRoot(k) || k.isPrefixOf(t.halfkey, false) {
		return nil
	}
	// A child of t is being subtracted
	if t.halfkey.isPrefixOf(k, false) {
		child := t.child(k.bit(t.halfkey.len))
		if *child != nil {
			*child = (*child).subtractKey(k.rest(t.halfkey.len))
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
func (t *tree[T]) subtractTree(o *tree[T]) *tree[T] {
	if o.hasEntry {
		// This whole branch is being subtracted; no need to traverse further
		if o.key.isPrefixOf(t.halfkey, false) {
			return nil
		}
		// A descendant of t is being subtracted
		if t.halfkey.isPrefixOf(o.key, false) {
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

func (t *tree[T]) isEmpty() bool {
	return t.halfkey.isZero() && t.left == nil && t.right == nil
}

// newParent returns a new node with halfkey h whose sole child is t.
//
// t and h must be in the same partition.
func (t *tree[T]) newParent(h halfkey) *tree[T] {
	if t.halfkey.len <= 64 != h.len <= 64 {
		panic("t.halfkey and s are in different partitions")
	}
	t.halfkey.offset = h.len
	parent := newTree[T](h).setChild(t)
	return parent
}

// mergeTree modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
// TODO: lots of duplicated code here
func (t *tree[T]) mergeTree(o *tree[T]) *tree[T] {
	// If o is empty, then the union is just t
	if o.isEmpty() {
		return t
	}

	if t.halfkey.equalFromRoot(o.key) {
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

	common := t.halfkey.commonPrefixLen(o.key)
	switch {
	// t.halfkey is a prefix of o.key
	case common == t.halfkey.len:
		// Traverse t in the direction of o
		tChildFollow := t.child(o.key.bit(t.halfkey.len))
		if *tChildFollow == nil {
			*tChildFollow = o.copy()
			(*tChildFollow).key.offset = t.halfkey.len
		} else {
			*tChildFollow = (*tChildFollow).mergeTree(o)
		}
		return t
	// o.key is a prefix of t.halfkey
	case common == o.key.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.newParent(o.key).setValueFrom(o).mergeTree(o)
	// Neither is a prefix of the other
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value.
		return t.newParent(t.halfkey.truncated(common)).setChild(
			newTree[T](o.key.rest(common)).setValueFrom(o),
		)
	}
}

func (t *tree[T]) intersectTreeImpl(
	o *tree[T],
	tPathHasEntry, oPathHasEntry bool,
) *tree[T] {

	// If o is an empty tree, then any intersection with it is also empty
	if o.isEmpty() {
		return &tree[T]{}
	}

	if t.halfkey.equalFromRoot(o.key) {
		// Consider t and o themselves.
		// If there is no entry in o at t.halfkey or above it, then remove t's
		// entry.
		//
		// TODO should this be t.remove(t.halfkey)? Could we end up with an
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

	common := t.halfkey.commonPrefixLen(o.key)
	switch {
	// t.halfkey is a prefix of o.key
	case common == t.halfkey.len:
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
		tChildFollow, tChildRemove := t.children(o.key.bit(common))

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

	// o.key is a prefix of t.halfkey
	case common == o.key.len:
		// o forks in the middle of t.halfkey. Similar to above.
		oChildFollow := o.child(t.halfkey.bit(common))

		// Traverse o in the direction of t.halfkey.
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
func (t *tree[T]) intersectTree(o *tree[T]) *tree[T] {
	return t.intersectTreeImpl(o, false, false)
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *tree[T]) insertHole(k halfkey, v T) *tree[T] {
	switch {
	// Removing t itself (no descendants will receive v)
	case t.halfkey.equalFromRoot(k):
		return nil
	// k is a descendant of t; start digging a hole to k
	case t.halfkey.isPrefixOf(k, false):
		t.clearValue()
		// Create a new sibling to receive v if needed, then continue traversing
		bit := k.bit(t.halfkey.len)
		child, sibling := t.children(bit)
		if *sibling == nil {
			*sibling = newTree[T](t.halfkey.next((^bit) & 1)).setValue(v)
		}
		*child = newTree[T](t.halfkey.next(bit)).insertHole(k, v)
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
func (t *tree[T]) walk(path halfkey, fn func(*tree[T]) bool) {
	// Follow provided path directly until it's exhausted
	n := t
	for n != nil && n.halfkey.len <= path.len {
		if !n.halfkey.isZero() {
			if fn(n) {
				return
			}
		}
		n = *(n.child(path.bit(n.halfkey.commonPrefixLen(path))))
	}

	if n == nil {
		return
	}

	// After path is exhausted, visit all children
	var st stack[*tree[T]]
	var stop bool
	st.Push(n)
	for !st.IsEmpty() {
		stop = false
		if n = st.Pop(); n == nil {
			continue
		}
		if !n.halfkey.isZero() {
			stop = fn(n)
		}
		if n.halfkey.len < 128 && !stop {
			st.Push(n.right)
			st.Push(n.left)
		}
	}
}

// pathNext returns the child of t which is next in the traversal of the
// specified path.
func (t *tree[T]) pathNext(path halfkey) *tree[T] {
	if path.bit(t.halfkey.len) == bitR {
		return t.right
	}
	return t.left
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T]) get(k halfkey) (val T, ok bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if n.halfkey.len >= k.len {
			if n.halfkey.equalFromRoot(k) && n.hasEntry {
				val, ok = n.value, true
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *tree[T]) contains(k halfkey) (ret bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if ret = n.halfkey.equalFromRoot(k) && n.hasEntry; ret {
			break
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
func (t *tree[T]) encompasses(k halfkey, strict bool) (ret bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if ret = n.hasEntry && n.halfkey.isPrefixOf(k, strict); ret {
			break
		}
	}
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T]) rootOf(k halfkey, strict bool) (outKey keyhalf, val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.hasEntry && n.halfkey.isPrefixOf(k, strict) {
			outKey, val, ok = n.halfkey, n.value, true
			return true
		}
		return false
	})
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T]) parentOf(k halfkey, strict bool) (outKey keyhalf, val T, ok bool) {
	t.walk(k, func(n *tree[T]) bool {
		if n.hasEntry && n.halfkey.isPrefixOf(k, strict) {
			outKey, val, ok = n.halfkey, n.value, true
		}
		return false
	})
	return
}

// descendantsOf returns the sub-tree containing all descendants of the
// provided key. The key itself will be included if it has an entry in the
// tree, unless strict == true. descendantsOf returns an empty tree if the
// provided key is not in the tree.
func (t *tree[T]) descendantsOf(k halfkey, strict bool) (ret *tree[T]) {
	ret = &tree[T]{}
	t.walk(k, func(n *tree[T]) bool {
		if k.isPrefixOf(n.halfkey, false) {
			ret.halfkey = n.halfkey.rooted()
			ret.left = n.left
			ret.right = n.right
			if !(strict && n.halfkey.equalFromRoot(k)) {
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
func (t *tree[T]) ancestorsOf(k halfkey, strict bool) (ret *tree[T]) {
	ret = &tree[T]{}
	t.walk(k, func(n *tree[T]) bool {
		if !n.halfkey.isPrefixOf(k, false) {
			return true
		}
		if n.hasEntry && !(strict && n.halfkey.equalFromRoot(k)) {
			ret.insert(n.halfkey, n.value)
		}
		return false
	})
	return
}

// filter updates t to include only the keys encompassed by o.
//
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
func (t *tree[T]) filter(o *tree[bool]) {
	remove := make([]halfkey, 0)
	t.walk(halfkey{}, func(n *tree[T]) bool {
		if !o.encompasses(n.halfkey, false) {
			remove = append(remove, n.halfkey)
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
func (t *tree[T]) filterCopy(o *tree[bool]) *tree[T] {
	ret := &tree[T]{}
	t.walk(halfkey{}, func(n *tree[T]) bool {
		if n.hasEntry && o.encompasses(n.halfkey, false) {
			ret = ret.insert(n.halfkey, n.value)
		}
		return false
	})
	return ret
}

// overlapsKey reports whether any key in t overlaps k.
func (t *tree[T]) overlapsKey(k halfkey) bool {
	var ret bool
	t.walk(k, func(n *tree[T]) bool {
		if !n.hasEntry {
			return false
		}
		if n.halfkey.isPrefixOf(k, false) || k.isPrefixOf(n.halfkey, false) {
			ret = true
			return true
		}
		return false
	})
	return ret
}
