package netipds

// tree is a binary radix tree.
//
// A valid tree has a non-nil root node having key.length == 0.
//
// The root node may have an entry (this enables natural support for 0.0.0.0/0
// and ::0/0).
type tree[T any, B keybits[B]] struct {
	key      key[B]
	hasEntry bool
	value    T
	left     *tree[T, B]
	right    *tree[T, B]
}

// newTree returns a new tree with the provided key.
func newTree[T any, B keybits[B]](k key[B]) *tree[T, B] {
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
	switch common {
	// Inserting at a descendant; recurse into the appropriate child
	case t.key.len:
		child := t.child(k.Bit(t.key.len))
		if *child == nil {
			*child = newTree[T](k.Rest(t.key.len)).setValue(v)
		}
		*child = (*child).insert(k, v)
		return t
	// Inserting at a prefix of t.key; create a new parent node with t as its
	// sole child
	case k.len:
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
		t.clearValue()
		switch {
		// No children (deleting a leaf node)
		case t.left == nil && t.right == nil:
			return t.nilOrEmptyRoot()
		// Root node: we must not replace the root node of the tree with a
		// non-zero-key node. If we did, then [tree.nilOrEmptyRoot] would not
		// be an effective guard against returning nil to users.
		case t.isRoot():
			return t
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
		if child := t.child(k.Bit(t.key.len)); *child != nil {
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
	// t is equal to, or a child of, the subtracted key; all of t will be removed
	// TODO EqualFromRoot call is now redundant
	if t.key.EqualFromRoot(k) || k.IsPrefixOf(t.key) {
		return t.nilOrEmptyRoot()
	}
	// A descendant of t is being subtracted
	if t.key.IsPrefixOf(k) {
		child := t.child(k.Bit(t.key.len))
		if *child != nil {
			*child = (*child).subtractKey(k.Rest(t.key.len))
		} else {
			t.insertHole(k, t.value, t.hasEntry)
		}
		if t.right == nil && t.left == nil && !t.hasEntry {
			return t.nilOrEmptyRoot()
		}
	}
	return t
}

// subtractTree removes all entries from t that have counterparts in o. If a
// child of t is removed, then new nodes may be created to fill in the gaps
// around the removed node.
func (t *tree[T, B]) subtractTree(o *tree[T, B]) *tree[T, B] {
	if t == nil || t.isEmpty() || o == nil || o.isEmpty() {
		return t
	}
	var zero T
	return t.subtractTreeImpl(o, zero, false)
}

func (t *tree[T, B]) subtractTreeImpl(o *tree[T, B], inheritedVal T, hasInherited bool) *tree[T, B] {
	if t == nil || o == nil || o.isEmpty() {
		return t
	}

	curVal := inheritedVal
	curHasVal := hasInherited
	if t.hasEntry {
		curHasVal = true
		curVal = t.value
	}

	switch {
	case o.key.EqualFromRoot(t.key):
		if o.hasEntry {
			return t.nilOrEmptyRoot()
		}
		t.shatter(curVal, curHasVal)
		for _, bit := range [2]bit{bitL, bitR} {
			oChild := o.child(bit)
			if *oChild == nil {
				continue
			}
			childPtr := t.child(bit)
			t.ensureChildForBit(childPtr, curHasVal, curVal, bit)
			if *childPtr == nil {
				continue
			}
			*childPtr = (*childPtr).subtractTreeImpl(*oChild, curVal, curHasVal)
		}
		return t.normalize()

	case o.key.IsPrefixOf(t.key):
		if o.hasEntry {
			return t.nilOrEmptyRoot()
		}
		bit := t.key.Bit(o.key.len)
		next := o.child(bit)
		if *next == nil {
			return t
		}
		return t.subtractTreeImpl(*next, curVal, curHasVal)

	case t.key.IsPrefixOf(o.key):
		t.shatter(curVal, curHasVal)
		bit := o.key.Bit(t.key.len)
		childPtr := t.child(bit)
		t.ensureChildForBit(childPtr, curHasVal, curVal, bit)
		if *childPtr == nil {
			return t.normalize()
		}
		*childPtr = (*childPtr).subtractTreeImpl(o, curVal, curHasVal)
		return t.normalize()

	default:
		return t
	}
}

// shatter materializes the entire key space under t so that subtractTree can
// remove specific subranges without losing everything.
//   - If t has an entry, clear it and treat its value as the inherited value.
//   - Ensure both children exist, inheriting that value.
//
// After this runs, the space beneath t is explicit and ready to be carved.
func (t *tree[T, B]) shatter(val T, hasVal bool) {
	if t == nil {
		return
	}
	if t.hasEntry {
		val = t.value
		hasVal = true
	}
	if !hasVal {
		return
	}
	t.hasEntry = false
	t.ensureChildForBit(t.child(bitL), true, val, bitL)
	t.ensureChildForBit(t.child(bitR), true, val, bitR)
}

// ensureChildForBit guarantees that the child referenced by childPtr starts at
// exactly one bit beyond t.key.
//   - If the pointer is nil and we have an inherited value, creates a new child
//     covering that single bit (shallowest possible).
//   - If the pointer already exists but its compressed edge skips over that bit,
//     splices in an intermediate parent so a node exists at the desired prefix.
//
// In both cases, the child ends up at the correct prefix and optionally inherits
// the provided value.
func (t *tree[T, B]) ensureChildForBit(childPtr **tree[T, B], hasVal bool, val T, bit bit) {
	if childPtr == nil {
		return
	}
	target := t.key.Next(bit)
	if *childPtr == nil {
		if !hasVal {
			return
		}
		*childPtr = newTree[T](target).setValue(val)
		return
	}
	// target is a prefix of the existing child's key. If the child already
	// starts at target, keep it; otherwise splice a new parent at target with
	// the existing child beneath it.
	child := *childPtr
	if child.key.len != target.len {
		parent := newTree[T](target)
		if child.key.Bit(target.len) == bitL {
			parent.left = child
		} else {
			parent.right = child
		}
		child.key.offset = target.len
		*childPtr = parent
	}
	if hasVal {
		(*childPtr).setValue(val)
	}
}

// normalize compresses t, collapsing redundant nodes, preserving the root
// invariant and reusing existing children when only one remains.
func (t *tree[T, B]) normalize() *tree[T, B] {
	if t == nil {
		return nil
	}
	if t.hasEntry {
		return t
	}
	switch {
	case t.left == nil && t.right == nil:
		return t.nilOrEmptyRoot()
	case t.left != nil && t.right == nil:
		t.left.key.offset = t.key.offset
		return t.left
	case t.left == nil && t.right != nil:
		t.right.key.offset = t.key.offset
		return t.right
	default:
		return t
	}
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *tree[T, B]) insertHole(k key[B], v T, tPathHasEntry bool) *tree[T, B] {
	switch {

	// Removing t itself (no descendants will receive v)
	case t.key.EqualFromRoot(k):
		return t.nilOrEmptyRoot()

	// k is a descendant of t; start digging a hole to k
	case t.key.IsPrefixOf(k):
		t.clearValue()
		bit := k.Bit(t.key.len)
		child, sibling := t.children(bit)

		// If there's no child in the direction of k and we're not currently
		// under an entry, then there is nothing to do. If we were under an
		// entry, then we would need to create new entries around the k hole.
		if *child == nil && !tPathHasEntry {
			return t
		}

		// Create a new sibling to receive v if needed, then continue traversing
		if *sibling == nil {
			*sibling = newTree[T](t.key.Next(!bit)).setValue(v)
		}

		// (child could be nil if we were carving a hole out of an entry)
		if *child == nil {
			*child = newTree[T](t.key.Next(bit))
		}

		// Continue digging hole
		*child = (*child).insertHole(k, v, t.hasEntry || tPathHasEntry)

	// k is an ancestor of t; remove t's entire branch
	case k.IsPrefixOf(t.key):
		return t.nilOrEmptyRoot()

	default:
		// Nothing to do
	}

	return t
}

// isEmpty returns true if t is a completely empty tree (no entry and no
// children)
func (t *tree[T, B]) isEmpty() bool {
	return !t.hasEntry && t.left == nil && t.right == nil
}

// newParent returns a new node with key k whose sole child is t.
func (t *tree[T, B]) newParent(k key[B]) *tree[T, B] {
	t.key.offset = (k.len)
	parent := newTree[T](k).setChild(t)
	return parent
}

// mergeTree modifies t so that it is the union of the entries of t and o.
func (t *tree[T, B]) mergeTree(o *tree[T, B]) *tree[T, B] {
	// If o is empty, then the union is just t
	if o.isEmpty() {
		return t
	}

	if t.key.EqualFromRoot(o.key) {
		if !t.hasEntry {
			t.setValueFrom(o)
		}

		for _, bit := range [2]bit{bitL, bitR} {
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
	switch common {
	// t.key is a prefix of o.key
	case t.key.len:
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
	case o.key.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.newParent(o.key).setValueFrom(o).mergeTree(o)
	// Neither is a prefix of the other
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value. We need a full copy of o in order to preserve all
		// of its children.
		oCopy := o.copy()
		oCopy.key = o.key.Rest(common)
		return t.newParent(t.key.Truncated(common)).setChild(oCopy)
	}
}

// intersectTree returns the intersection of t and o as a new tree.
// An entry is included in the result iff it is encompassed by both t and o.
func (t *tree[T, B]) intersectTree(o *tree[T, B]) *tree[T, B] {
	var path key[B]
	var result *tree[T, B] = &tree[T, B]{}

	// Include every entry in t that o encompasses
	t.walk(path, func(n *tree[T, B]) bool {
		if n.hasEntry && o.encompasses(n.key) {
			result = result.insert(n.key.Rooted(), n.value)
		}
		return false
	})

	// Include every entry in o that t encompasses
	o.walk(path, func(n *tree[T, B]) bool {
		if n.hasEntry && t.encompasses(n.key) {
			result = result.insert(n.key.Rooted(), n.value)
		}
		return false
	})

	return result
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
		if fn(n) {
			return
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
		stop = fn(n)
		if n.key.len < stackMaxDepth && !stop {
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
	u128 := k.content.Uint128()
	for n := t; n != nil; n = n.pathNext(u128) {
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
	u128 := k.content.Uint128()
	for n := t; n != nil; n = n.pathNext(u128) {
		if ret = n.key.EqualFromRoot(k) && n.hasEntry; ret {
			break
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses or is equal to the provided key.
func (t *tree[T, B]) encompasses(k key[B]) (ret bool) {
	u128 := k.content.Uint128()
	for n := t; n != nil; n = n.pathNext(u128) {
		if ret = n.hasEntry && n.key.IsPrefixOf(k); ret {
			break
		}
	}
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
func (t *tree[T, B]) rootOf(k key[B]) (outKey key[B], val T, ok bool) {
	u128 := k.content.Uint128()
	for n := t; n != nil; n = n.pathNext(u128) {
		if ok = n.hasEntry && n.key.IsPrefixOf(k); ok {
			return n.key, n.value, ok
		}
	}
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
func (t *tree[T, B]) parentOf(k key[B]) (outKey key[B], val T, ok bool) {
	u128 := k.content.Uint128()
	for n := t; n != nil; n = n.pathNext(u128) {
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
// ancestorsOf returns an empty tree if k has no ancestors in the tree.
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
//
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
//
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

// isRoot returns true iff t is a root node.
func (t *tree[T, B]) isRoot() bool {
	return t.key.IsZero()
}

// nilOrEmptyRoot returns nil unless t is the root node, in which case it
// returns a new empty root node. This is useful in recursive functions that
// would otherwise return nil to remove a node: it ensures users of tree never
// receive a nil.
//
// This relies on an invariant: tree's root node must always have a zero-length
// key. Otherwise, nilOrEmptyRoot will fail to identify it as the root node,
// and could return nil to a user.
//
// Note: this method should be the only method of tree containing the statement
// `return nil` (unless the receiver is known to be nil).
func (t *tree[T, B]) nilOrEmptyRoot() *tree[T, B] {
	if t.isRoot() {
		return &tree[T, B]{}
	}
	return nil
}
