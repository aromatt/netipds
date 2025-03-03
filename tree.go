package netipds

import (
	"fmt"
)

// tree is a compressed binary radix tree supporting 128-bit keys.
//
// The tree is partitioned depth-wise into halves: "hi" (most-significant 64
// bits) and "lo" (least-significant 64 bits). Each node owns a "halfkey" which
// is aligned at offset 0 or 64 in the full key space.
type tree[T any] struct {
	halfkey  halfkey
	hasEntry bool
	value    T
	left     *tree[T]
	right    *tree[T]
}

/*

Helpful Invariants

  1. Each halfkey's offset and len must be in the same partition, i.e. both
     must lie in the range [0, 63] or both must lie in the range [64, 127].

  2. A node in the lo partition must have a parent that either (a) is also in
     the lo partition or (b) is in the hi partition and has len == 64 (in this
	 case the lo node must have offset == 64, and we call the parent a "bridge
     node").

  3. A node in the lo partition must have either (a) a parent in the lo partition
     or (b) offset == 64.

*/

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

// adopt sets one of t's children to n, if it isn't already set, choosing
// which child based on the first bit of n (n.halfkey.offset). A provided nil
// is ignored.
func (t *tree[T]) adopt(n *tree[T]) *tree[T] {
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

// childOrCreate returns the child of t that follows the path of k.
//
// If no such child exists, then a new child owning k is created and attached
// to t (via a bridge node if k crosses partitions).
//
// The bool is true iff the child already existed.
func (t *tree[T]) childOrCreate(k key) (**tree[T], bool) {
	c := t.child(k.bit(t.halfkey.len))
	if *c != nil {
		return c, true
	}

	var kHalf halfkey
	kHi, kLo := k.halves()

	// Handle partition scenarios
	switch {

	// k crosses partition boundary (starts in hi and ends in lo)
	case !kHi.isZero() && !kLo.isZero():
		*c = newTree[T](kHi)
		c = (*c).child(k.bit(64))
		kHalf = kLo

	// k is contained in hi partition
	case !kHi.isZero():
		kHalf = kHi

	// k is contained in lo partition
	case !kLo.isZero():
		kHalf = kLo

	default:
		panic("unreachable")
	}

	*c = newTree[T](kHalf)
	return c, false
}

// addChildAt creates a new child node to own k, attaches it at c (via a new
// bridge node, if k crosses the partition boundary), and returns it.
//
// We implement as a function instead of a method because if a bridge node is
// required, the caller needs two things to happen: the bridge node must be
// attached to the tree, and the child must be returned for further traversal
// or modification.
func addChildAt[T any](c **tree[T], k key) *tree[T] {
	var kHalf halfkey
	kHi, kLo := k.halves()

	switch {
	// k crosses partition boundary (starts in hi and ends in lo)
	case !kHi.isZero() && !kLo.isZero():
		*c = newTree[T](kHi)
		c = (*c).child(k.bit(64))
		kHalf = kLo
	// k is contained in hi partition
	case !kHi.isZero():
		kHalf = kHi
	// k is contained in lo partition
	case !kLo.isZero():
		kHalf = kLo
	default:
		panic("unreachable")
	}

	*c = newTree[T](kHalf)
	return *c
}

// insert inserts value v at key k with path compression.
func (t *tree[T]) insert(k key, v T) *tree[T] {
	println("\ninsert:\nt:", t.String(), "\nk:", k.String())
	// Inserting at t itself
	if t.halfkey.keyEqualFromRoot(k) {
		return t.setValue(v)
	}

	common := t.halfkey.keyHalfCommonPrefixLen(k)
	switch {

	// Inserting at a descendant; recurse into the appropriate child.
	//
	// We know that k must be strictly longer than t, because (1) the check
	// above ruled out the possibility that k == t and (2) t is itself the
	// common prefix.
	case common == t.halfkey.len:
		// Select the child of t to recurse into or create.
		//   t: 000
		//   k: 0001
		//         ^ this bit determines which child of t to select
		//
		// TODO: What about this?
		//   t: 0000
		//   k: 0000 1
		if child, ok := t.childOrCreate(k.rest(t.halfkey.len)); ok {
			*child = (*child).insert(k, v)
		} else {
			(*child).setValue(v)
		}
		return t

	// Inserting at a prefix of t, e.g.:
	//   t: 0000
	//   k: 00
	// Create a new parent node with t as its sole child.
	//
	// We know that t must be strictly longer than k, because (1) the check
	// above ruled out the possibility that k == t and (2) k is itself the
	// common prefix.
	//
	// We also know that t and k must be in the same partition. We can't have:
	//   t: 0000 0
	//   k: 00
	// because t must have a parent in the hi partition which would handle the
	// insertion first.
	case common == k.len:
		println("inserting at a prefix of t\nt:", t.String(), "\nk:", k.String())
		// TODO remove?
		if (t.halfkey.len > 64) != (k.len > 64) {
			panic("unreachable")
		}
		return t.newParent(k.endHalf()).setValue(v)

	// Neither is a prefix of the other, e.g.:
	//   t: 000
	//   k: 001
	// Create a new parent at their common prefix having t and its new sibling
	// as children.
	//
	// We know that either t and k are in the same partition, or t is in hi and
	// k is in lo, e.g.
	//   t: 000
	//   k: 0010 0
	// In this case, their common prefix must be in hi.
	//
	// We can't have:
	//   t: 0000 0
	//   k: 001
	// because t must have a parent in the hi partition which would handle the
	// insertion first.
	//
	// TODO: What about this?
	//   t: 0000 0
	//   k: 0000 1
	// This looks nasty. But wait... this would be handled by the first case.
	default:
		println("\ncreating new parent\nt:", t.String(), "\nk:", k.String())
		p := t.newParent(t.halfkey.truncated(common))
		c, _ := p.childOrCreate(k.rest(common))
		(*c).setValue(v)
		return p
	}
}

/* HACK
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
*/

func (t *tree[T]) isEmpty() bool {
	return t.halfkey.isZero() && t.left == nil && t.right == nil
}

// newParent returns a new node to own k whose sole child is t (via a bridge
// node, if necessary). The new parent takes over ownership of t's first bits
// through k.len, and t.offset is modified accordingly.
func (t *tree[T]) newParent(h halfkey) *tree[T] {
	if (t.halfkey.len <= 64) != (h.len <= 64) {
		panic("t.halfkey and h are in different partitions")
	}
	t.halfkey.offset = h.len
	return newTree[T](h).adopt(t)
}

// insertParent returns a new node to own k whose sole child is t (via a bridge
// node, if necessary). The new parent takes over ownership of t's first bits
// through k.len, and t.offset is modified accordingly.
//func (t *tree[T]) insertParent(k key) *tree[T] {
//	//var kHalf halfkey
//	tRest := t.
//	tHi, tLo := tRest.halves()
//
//	switch {
//
//	// t ends in hi partition and k ends in lo partition, e.g.
//	//   t: 000
//	//   k: 0010 0
//	case k.len <= 64 && t.halfkey.len > 64:
//		t.halfkey.offset = k.len
//		parent = newTree[T](halfkey{t.halfkey.content, t.halfkey.offset, 64})
//		c = (*c).child(
//
//
//		kHalf = kLo
//	// t and k are both contained in hi partition
//	case !kHi.isZero():
//		kHalf = kHi
//	// t and k are both contained in lo partition
//	case !kLo.isZero():
//		kHalf = kLo
//	default:
//		panic("unreachable")
//	}
//	(*c).halfkey.offset = kHalf.len
//
//	return newTree[T](kHalf).adopt(*c)
//}

/* HACK
// mergeTree modifies t so that it is the union of the entries of t and o.
func (t *tree[T]) mergeTree(o *tree[T]) *tree[T] {
	// TODO: same problem as subtractTree; only makes sense for PrefixSets.
	// TODO: lots of duplicated code here

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

	// t is a prefix of o, e.g.
	//   t: 00
	//   o: 000
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

	// o is a prefix of t, e.g.
	//   t: 000
	//   o: 00
	case common == o.key.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.newParent(o.key).setValueFrom(o).mergeTree(o)

	// Neither is a prefix of the other, e.g.
	//   t: 0010
	//   o: 000
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value.
		return t.newParent(
			t.halfkey.truncated(common),
		).adopt(
			newTree[T](o.key.rest(common)).setValueFrom(o),
		)
	}
}

func (t *tree[T]) intersectTreeImpl(
	o *tree[T],
	tPathHasEntry, oPathHasEntry bool,
) *tree[T] {

	// If o is empty, then any intersection with it is also empty
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
func (t *tree[T]) intersectTree(o *tree[T]) *tree[T] {
	// TODO: same problem as subtractTree; only makes sense for PrefixSets.
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
		b := k.bit(t.halfkey.len)
		child, sibling := t.children(b)
		if *sibling == nil {
			//*sibling = newTree[T](t.halfkey.next((^bit) & 1)).setValue(v)
			addChildAt(sibling, t.halfkey.next((^b)&1)).setValue(v)
		}
		//*child = newTree[T](t.halfkey.next(bit)).insertHole(k, v)
		addChildAt(child, t.halfkey.next(b)).insertHole(k, v)
		//*child = newChild[T](t.halfkey.next(b), func(c *tree[T]) {
		//	c.insertHole(k, v)
		//})
		return t
	// Nothing to do
	default:
		return t
	}
}
*/

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
//
// For example:
//
//	t:    00
//	path: 0000
//	        ^-- bit == 0; return t.left
func (t *tree[T]) pathNext(path key) *tree[T] {
	if path.bit(t.halfkey.len) == bitR {
		return t.right
	}
	return t.left
}

// get returns the value associated with the exact key provided, if it exists.
func (t *tree[T]) get(k key) (val T, ok bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if n.halfkey.len >= k.len {
			// TODO when crossing to lo partition, do we need to check the
			// bridge node for equality to the hi half of k? I think so,
			// but maybe not.
			if n.halfkey.keyEndEqualFromRoot(k) && n.hasEntry {
				val, ok = n.value, true
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *tree[T]) contains(k key) (ret bool) {
	for n := t.pathNext(k); n != nil; n = n.pathNext(k) {
		if ret = n.halfkey.keyEndEqualFromRoot(k) && n.hasEntry; ret {
			break
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
/* HACK
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
func (t *tree[T]) rootOf(k halfkey, strict bool) (outKey halfkey, val T, ok bool) {
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
func (t *tree[T]) parentOf(k halfkey, strict bool) (outKey halfkey, val T, ok bool) {
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
// tree, unless strict == true. descendantsOf returns the empty tree if the
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
// strict == true. ancestorsOf returns the empty tree if key has no ancestors in
// the tree.
/* HACK
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
*/

// filter updates t to include only the keys encompassed by o.
//
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
/* HACK
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
*/

// filterCopy returns a recursive copy of t that includes only keys that are
// encompassed by o.
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
// TODO: does it make sense to have both this method and filter()?
/* HACK
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
*/
