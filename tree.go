package netipds

import (
	"fmt"
)

type nodeRef int

type node struct {
	key key
	// sr: we still have hasEntry because PrefixSet doesn't have values but it
	// has entries
	hasEntry bool
	left     nodeRef
	right    nodeRef
}

// tree is a binary radix tree supporting 128-bit keys (see key.go).
//
// The tree is compressed by default, however it supports uncompressed
// insertion via insertLazy(). This can be much faster than insert() and works
// well with netipds's intended usage pattern (build a collection with a
// builder type, then generate an immutable version). After lazy insertions,
// the tree can be compressed using the compress() method.
type tree[T any] struct {
	// The absolute root node is always at index 0.
	nodes []node
	// Values are indexed by the node's index in the nodes slice.
	values map[nodeRef]T
	// cur is a reference to some node in the tree.
	// This is the node from which traversals will start.
	// TODO
	cur nodeRef
}

// newNode creates a new node in t with the provided key.
func (t *tree[T]) newNode(k key) nodeRef {
	n := node{key: k}
	t.nodes = append(t.nodes, n)
	return nodeRef(len(t.nodes))
}

// newTree returns a new tree.
func newTree[T any]() *tree[T] {
	return &tree[T]{
		nodes:  []node{node{}},
		values: map[nodeRef]T{},
	}
}

// setValue sets n's value to v and returns n.
func (t *tree[T]) setValue(n nodeRef, v T) nodeRef {
	t.values[n] = v
	return n
}

// clearEntry removes the entry and value from t.
func (t *tree[T]) clearEntry(n nodeRef) {
	var zeroVal T
	t.values[n] = zeroVal
	t.nodes[n].hasEntry = false
}

// setValueFrom copies one node's value to another within the same tree.
func (t *tree[T]) setValueFrom(n, o nodeRef) {
	if t.nodes[o].hasEntry {
		t.setValue(n, t.values[o])
	}
}

// childAt returns the index of the child of n specified by b.
func (t *tree[T]) childAt(n nodeRef, b bit) nodeRef {
	if b == bitR {
		return t.nodes[n].right
	}
	return t.nodes[n].left
}

// children returns the indices of n's children in the order indicated by
// whichFirst.
func (t *tree[T]) children(n nodeRef, whichFirst bit) (id1, id2 nodeRef) {
	if whichFirst == bitR {
		return t.nodes[n].right, t.nodes[n].left
	}
	return t.nodes[n].left, t.nodes[n].right
}

// setChildAt assigns o as the child of n specified by b.
func (t *tree[T]) setChildAt(n, o nodeRef, b bit) {
	if b == bitR {
		t.nodes[n].right = o
	}
	t.nodes[n].left = o
}

// setChild sets o as the appropriate child of n if the child spot isn't
// already taken, choosing which child position based on the bit at the
// beginning of o's key segment (i.e. at key.offset). A provided 0 is ignored.
// TODO: method names are confusing re: whether they move the cursor or not
func (t *tree[T]) setChild(n, o nodeRef) nodeRef {
	if o == 0 {
		return o
	}
	oKey := t.nodes[o].key
	b := oKey.bit(oKey.offset)
	if t.childAt(n, b) == 0 {
		t.setChildAt(n, o, b)
	}
	return o
}

func (t *tree[T]) cursor() treeCursor[T] {
	return treeCursor[T]{t, 0}
}

func (t *tree[T]) String() string {
	return t.cursor().stringImpl("", "", false)
}

// treeCursor is used for recursive methods that operate on a tree.
// It includes a nodeRef to track the current position in the tree.
type treeCursor[T any] struct {
	*tree[T]
	node nodeRef
}

// tc2 is used for traversing two trees simultaneously.
type tc2[T any] [2]treeCursor[T]

// Node returns the current node.
func (t treeCursor[T]) Node() node {
	return t.nodes[t.node]
}

// SetNode replaces the node referred to by the cursor with the provided one.
func (t treeCursor[T]) SetNode(n node) {
	t.nodes[t.node] = n
}

// Key returns the key of the current node.
func (t treeCursor[T]) Key() key {
	return t.Node().key
}

func (t treeCursor[T]) Value() (T, bool) {
	val, ok := t.values[t.node]
	return val, ok
}

// Left returns a cursor positioned at the current node's left child if it
// exists. If it doesn't exist, Left returns (t, false).
func (t treeCursor[T]) Left() (treeCursor[T], bool) {
	left := t.tree.nodes[t.node].left
	if left == 0 {
		return t, false
	}
	return treeCursor[T]{tree: t.tree, node: t.tree.nodes[t.node].left}, true
}

// Right returns a cursor positioned at the current node's right child if it
// exists. If it doesn't exist, Right returns (t, false).
func (t treeCursor[T]) Right() (treeCursor[T], bool) {
	right := t.tree.nodes[t.node].right
	if right == 0 {
		return t, false
	}
	return treeCursor[T]{tree: t.tree, node: t.tree.nodes[t.node].right}, true
}

// ChildAt returns a cursor positioned at the child (selected by b) of the
// current node.
// TODO: do we need Left and Right if we have this?
func (t treeCursor[T]) ChildAt(b bit) (treeCursor[T], bool) {
	child := t.tree.childAt(t.node, b)
	if child == 0 {
		return t, false
	}
	return treeCursor[T]{t.tree, child}, true
}

// NewChildAt creates a zero-valued node as the child (selected by b) of the
// current node.
func (t treeCursor[T]) NewChildAt(b bit) treeCursor[T] {
	return t.SetChild(t.newNode(key{}))
}

// SetChild sets o as the appropriate child of the current node and returns a
// cursor positioned at the new node.
func (t treeCursor[T]) SetChild(o nodeRef) treeCursor[T] {
	return treeCursor[T]{t.tree, t.tree.setChild(t.node, o)}
}

// SetChildAt sets o as the child of the current node selected by b.
func (t treeCursor[T]) SetChildAt(o nodeRef, b bit) treeCursor[T] {
	t.setChildAt(t.node, o, b)
	return t
}

// AddChild adds a new node with key k as a child of the current node and
// returns a cursor positioned at the new node.
func (t treeCursor[T]) AddChild(k key) treeCursor[T] {
	return treeCursor[T]{t.tree, t.tree.setChild(t.node, t.newNode(k))}
}

// SetValue updates the value of the current node.
func (t treeCursor[T]) SetValue(v T) treeCursor[T] {
	t.tree.setValue(t.node, v)
	return t
}

func (t treeCursor[T]) SetOffset(offset uint8) treeCursor[T] {
	t.tree.nodes[t.node].key.offset = offset
	return t
}

// Copy returns a copy of t.
//
// Note: values are copied using regular assignment, so if the values are
// pointers, the copied tree will share references with the original.
//
// Note: this has the side effect of garbage collecting the nodes slice and
// values map (for the copy, not the original).
func (t *tree[T]) Copy() *tree[T] {
	ret := newTree[T]()
	s := stack[tc2[T]]{}
	s.Push(tc2[T]{t.cursor(), ret.cursor()})
	for !s.IsEmpty() {
		c := s.Pop()
		src, dst := c[0], c[1]
		dst.SetNode(node{key: src.Key(), hasEntry: src.Node().hasEntry})
		// TODO: adjust this if we support a value-less tree
		if val, ok := src.Value(); ok {
			dst.SetValue(val)
		}
		if left, ok := src.Left(); ok {
			s.Push(tc2[T]{left, dst.NewChildAt(bitL)})
		}
		if right, ok := src.Right(); ok {
			s.Push(tc2[T]{right, dst.NewChildAt(bitR)})
		}
	}
	return ret
}

// TODO: get rid of hideVal if possible
// TODO: if hideVal, still distinguish entries from non-entries
func (t treeCursor[T]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	nn := t.Node()
	if hideVal || !nn.hasEntry {
		ret = fmt.Sprintf("%s%s%s\n", indent, pre, nn.key.StringRel())
	} else {
		ret = fmt.Sprintf("%s%s%s: %v\n", indent, pre, nn.key.StringRel(),
			t.tree.values[t.node])
	}
	if left, ok := t.Left(); ok {
		ret += left.stringImpl(indent+"  ", "L:", hideVal)
	}
	if right, ok := t.Right(); ok {
		ret += right.stringImpl(indent+"  ", "R:", hideVal)
	}
	return ret
}

// Size returns the number of entries in t.
// TODO: keep track of this instead of calculating it lazily
func (t treeCursor[T]) Size() (size int) {
	if t.Node().hasEntry {
		size += 1
	}
	if left, ok := t.Left(); ok {
		size += left.Size()
	}
	if right, ok := t.Right(); ok {
		size += right.Size()
	}
	return
}

// Insert inserts value v at key k with path compression and moves the cursor
// to the inserted node.
func (t treeCursor[T]) Insert(k key, v T) treeCursor[T] {
	tKey := t.Key()
	// Inserting at the current node itself
	if tKey.equalFromRoot(k) {
		return t.SetValue(v)
	}

	com := tKey.commonPrefixLen(k)
	switch {
	// Inserting at a descendant; recurse into the appropriate child
	case com == tKey.len:
		child, ok := t.ChildAt(k.bit(tKey.len))
		if !ok {
			child = t.AddChild(k.rest(tKey.len)).SetValue(v)
		}
		child.Insert(k, v)
		return t
	// Inserting at a prefix of tKey; create a new parent node with t as its
	// sole child
	case com == k.len:
		return t.InsertParent(k).SetValue(v)
	// Neither is a prefix of the other; create a new parent at their common
	// prefix with children t and its new sibling
	default:
		parent := t.InsertParent(tKey.trunc(com))
		parent.AddChild(k.rest(com)).SetValue(v)
		return parent
	}
}

// InsertLazy inserts value v at key k without path compression.
func (t treeCursor[T]) InsertLazy(k key, v T) treeCursor[T] {
	tKey := t.Key()
	switch {
	// Inserting at t itself
	case tKey.equalFromRoot(k):
		return t.SetValue(v)
	// Inserting at a descendant
	case tKey.commonPrefixLen(k) == tKey.len:
		bit := k.bit(tKey.len)
		child, ok := t.ChildAt(bit)
		if !ok {
			child = t.AddChild(tKey.next(bit))
		}
		child.InsertLazy(k, v)
		return t
	// Nothing to do
	default:
		return t
	}
}

// compress performs path compression on tree t.
// TODO: test this, I don't think it was correct before the slice refactor
func (t treeCursor[T]) compress() treeCursor[T] {
	left, leftOk := t.Left()
	right, rightOk := t.Right()
	switch {
	case !leftOk && !rightOk:
		return t
	case leftOk:
		right.SetOffset(t.Key().offset)
		return right.compress()
	case rightOk:
		left.SetOffset(t.Key().offset)
		return left.compress()
	default:
		return t
	}
}

// Remove removes the exact provided key from the tree, if it exists, and
// performs path compression.
func (t treeCursor[T]) Remove(k key) nodeRef {
	tKey := t.Key()
	switch {
	// Removing t itself
	case k.equalFromRoot(tKey):
		if t.Node().hasEntry {
			t.clearEntry(t.node)
		}
		left, leftOk := t.Left()
		right, rightOk := t.Right()
		switch {
		// No children (deleting a leaf node)
		case !leftOk && !rightOk:
			return 0 // 0 represents the absence of a node
		// Only one child; merge with it
		case !leftOk:
			right.SetOffset(tKey.offset)
			return right.node
		case !rightOk:
			left.SetOffset(tKey.offset)
			return left.node
		// t is a shared prefix node, so it can't be removed
		default:
			return t.node
		}
	// Removing a descendant of t; recurse into the appropriate child
	case tKey.isPrefixOf(k, false):
		bit := k.bit(tKey.len)
		if child, ok := t.ChildAt(bit); ok {
			// We need to use SetChildAt because the returned nodeRef may be 0
			t.SetChildAt(child.Remove(k), bit)
		}
		return t.node
	// Nothing to do
	default:
		return t.node
	}
}

// SubtractKey removes k and all of its descendants from the tree, leaving the
// remaining key space behind. If k is a descendant of t, then new nodes may be
// created to fill in the gaps around k.
func (t treeCursor[T]) SubtractKey(k key) nodeRef {
	tk := t.Key()
	// This whole branch is being subtracted; no need to traverse further
	if tk.equalFromRoot(k) || k.isPrefixOf(tk, false) {
		return 0
	}
	// A child of t is being subtracted
	if tk.isPrefixOf(k, false) {
		bit := k.bit(tk.len)
		child, ok := t.ChildAt(bit)
		if ok {
			// We need to use SetChildAt because the returned nodeRef may be 0
			t.SetChildAt(child.SubtractKey(k.rest(tk.len)), bit)
		} else {
			t.insertHole(k, t.value)
		}
		tn := t.Node()
		if tn.right == 0 && tn.left == 0 && !tn.hasEntry {
			return 0
		}
	}
	return t.node
}

// subtractTree removes all entries from t that have counterparts in o. If a
// child of t is removed, then new nodes may be created to fill in the gaps
// around the removed node.
//
// TODO: this method only makes sense in the context of a PrefixSet.
// "subtracting" a whole key-value entry from another isn't meaningful. So
// maybe we need two types of trees: value-bearing ones, and others that just
// have value-less entries.
func (t *node[T]) subtractTree(o *node[T]) *node[T] {
	if o.hasEntry {
		// This whole branch is being subtracted; no need to traverse further
		if o.key.isPrefixOf(t.key, false) {
			return nil
		}
		// A descendant of t is being subtracted
		if t.key.isPrefixOf(o.key, false) {
			t.insertHole(o.key, t.value)
		}
	}
	// Consider the children of both t and o
	for _, bit := range eachBit {
		tChild, oChild := t.childAt(bit), o.childAt(bit)
		if *oChild != nil {
			if *tChild == nil {
				tChild = &t
			}
			*tChild = (*tChild).subtractTree(*oChild)
		}
	}
	return t
}

// TODO: remove
func (t *node[T]) isEmpty() bool {
	return t.key.isZero() && t.left == nil && t.right == nil
}

// InsertParent inserts a new node with key k as the parent of the current node
// and returns a cursor pointing to the new parent.
func (t treeCursor[T]) InsertParent(k key) treeCursor[T] {
	t.nodes[t.node].key.offset = k.len
	return treeCursor[T]{t.tree, t.tree.setChild(t.newNode(k), t.node)}
}

// mergeTree modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
// TODO: lots of duplicated code here
func (t *node[T]) mergeTree(o *node[T]) *node[T] {
	// If o is empty, then the union is just t
	if o.isEmpty() {
		return t
	}

	if t.key.equalFromRoot(o.key) {
		if !t.hasEntry {
			t.setValueFrom(o)
		}

		for _, bit := range eachBit {
			tChild, oChild := t.childAt(bit), o.childAt(bit)
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

	com := t.key.commonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	case com == t.key.len:
		// Traverse t in the direction of o
		tChildFollow := t.childAt(o.key.bit(t.key.len))
		// sr: something like:
		// if tChildFollow == 0 {
		//     o.copyTo(t, n, o)
		if *tChildFollow == nil {
			*tChildFollow = o.copy()
			// o.copyTo(<tree>, t, o)
			(*tChildFollow).key.offset = t.key.len
		} else {
			*tChildFollow = (*tChildFollow).mergeTree(o)
		}
		return t
	// o.key is a prefix of t.key
	case com == o.key.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.InsertParent(o.key).setValueFrom(o).mergeTree(o)
	// Neither is a prefix of the other
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value.
		return t.InsertParent(t.key.trunc(com)).setChild(
			newNode[T](o.key.rest(com)).setValueFrom(o),
		)
	}
}

func (t *node[T]) intersectTreeImpl(
	o *node[T],
	tPathHasEntry, oPathHasEntry bool,
) *node[T] {

	// If o is an empty tree, then any intersection with it is also empty
	if o.isEmpty() {
		return &node[T]{}
	}

	if t.key.equalFromRoot(o.key) {
		// Consider t and o themselves.
		// If there is no entry in o at t.key or above it, then remove t's
		// entry.
		//
		// TODO should this be t.remove(t.key)? Could we end up with an
		// unnecessary prefix node?
		if t.hasEntry && !(o.hasEntry || oPathHasEntry) {
			t.clearEntry()
			// We need to remember that t had an entry here so that o's
			// descendants are kept
			tPathHasEntry = true
		}

		// Consider the children of t and o
		for _, bit := range eachBit {
			tChild, oChild := t.childAt(bit), o.childAt(bit)
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

	com := t.key.commonPrefixLen(o.key)
	switch {
	// t.key is a prefix of o.key
	case com == t.key.len:
		if t.hasEntry {
			if !oPathHasEntry {
				t.clearEntry()
			}
			// TODO: do we know enough at this point to avoid a general insert()?
			t = t.Insert(o.key, o.value)
		}

		// t forks in the middle of o.key. To take the intersection, we
		// need to traverse t toward o.key and prune the other child of t.
		//
		// The bit of o.key just after the common prefix determines which
		// of t's children to follow and which to remove.
		// e.g. t=00, o=000 -> follow left, remove right
		// sr:
		// - this could be natural to return uint64's
		tChildFollow, tChildRemove := t.children(o.key.bit(com))

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
	case com == o.key.len:
		// o forks in the middle of t.key. Similar to above.
		oChildFollow := o.childAt(t.key.bit(com))

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
func (t treeCursor[T]) intersectTree(o treeCursor[T]) *node[T] {
	return t.intersectTreeImpl(o, false, false)
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t *node[T]) insertHole(k key, v T) *node[T] {
	switch {
	// Removing t itself (no descendants will receive v)
	case t.key.equalFromRoot(k):
		return nil
	// k is a descendant of t; start digging a hole to k
	case t.key.isPrefixOf(k, false):
		t.clearEntry()
		// Create a new sibling to receive v if needed, then continue traversing
		bit := k.bit(t.key.len)
		child, sibling := t.children(bit)
		if *sibling == nil {
			*sibling = newNode[T](t.key.next((^bit) & 1)).setValue(v)
		}
		*child = newNode[T](t.key.next(bit)).insertHole(k, v)
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
func (t *node[T]) walk(path key, fn func(*node[T]) bool) {
	// Follow provided path directly until it's exhausted
	n := t
	for n != nil && n.key.len <= path.len {
		if !n.key.isZero() {
			if fn(n) {
				return
			}
		}
		n = *(n.childAt(path.bit(n.key.commonPrefixLen(path))))
	}

	if n == nil {
		return
	}

	// After path is exhausted, visit all children
	var st stack[*node[T]]
	var stop bool
	st.Push(n)
	for !st.IsEmpty() {
		stop = false
		if n = st.Pop(); n == nil {
			continue
		}
		if !n.key.isZero() {
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
func (t *node[T]) pathNext(path key) *node[T] {
	if path.bit(t.key.len) == bitR {
		return t.right
	}
	return t.left
}

// get returns the value associated with the exact key provided, if it exists.
func (t *node[T]) get(k key) (val T, ok bool) {
	for n := t; n != nil; n = n.pathNext(k) {
		if !n.key.isZero() && n.key.len >= k.len {
			if n.key.equalFromRoot(k) && n.hasEntry {
				val, ok = n.value, true
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t *node[T]) contains(k key) (ret bool) {
	for n := t; n != nil; n = n.pathNext(k) {
		if !n.key.isZero() {
			if ret = (n.key.equalFromRoot(k) && n.hasEntry); ret {
				break
			}
		}
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
func (t *node[T]) encompasses(k key, strict bool) (ret bool) {
	t.walk(k, func(n *node[T]) bool {
		ret = n.key.isPrefixOf(k, strict) && n.hasEntry
		if ret {
			return true
		}
		return false
	})
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *node[T]) rootOf(k key, strict bool) (outKey key, val T, ok bool) {
	t.walk(k, func(n *node[T]) bool {
		if n.key.isPrefixOf(k, strict) && n.hasEntry {
			outKey, val, ok = n.key, n.value, true
			return true
		}
		return false
	})
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *node[T]) parentOf(k key, strict bool) (outKey key, val T, ok bool) {
	t.walk(k, func(n *node[T]) bool {
		if n.key.isPrefixOf(k, strict) && n.hasEntry {
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
func (t *node[T]) descendantsOf(k key, strict bool) (ret *node[T]) {
	ret = &node[T]{}
	t.walk(k, func(n *node[T]) bool {
		if k.isPrefixOf(n.key, false) {
			ret.key = n.key.rooted()
			ret.left = n.left
			ret.right = n.right
			if !(strict && n.key.equalFromRoot(k)) {
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
func (t *node[T]) ancestorsOf(k key, strict bool) (ret *node[T]) {
	ret = &node[T]{}
	t.walk(k, func(n *node[T]) bool {
		if !n.key.isPrefixOf(k, false) {
			return true
		}
		if n.hasEntry && !(strict && n.key.equalFromRoot(k)) {
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
func (t *node[T]) filter(o *node[bool]) {
	remove := make([]key, 0)
	t.walk(key{}, func(n *node[T]) bool {
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
func (t *node[T]) filterCopy(o *node[bool]) *node[T] {
	ret := &node[T]{}
	t.walk(key{}, func(n *node[T]) bool {
		if n.hasEntry && o.encompasses(n.key, false) {
			ret = ret.insert(n.key, n.value)
		}
		return false
	})
	return ret
}

// overlapsKey reports whether any key in t overlaps k.
func (t *node[T]) overlapsKey(k key) bool {
	var ret bool
	t.walk(k, func(n *node[T]) bool {
		if !n.hasEntry {
			return false
		}
		if n.key.isPrefixOf(k, false) || k.isPrefixOf(n.key, false) {
			ret = true
			return true
		}
		return false
	})
	return ret
}
