package netipds

import (
	"fmt"
)

type nodeRef int

const absent = nodeRef(0)

// tree is a binary radix tree supporting 128-bit keys (see key.go).
type tree[B KeyBits[B], T any] struct {
	// Index 0 is occupied by the root node, and is also used as a sentinel
	// nodeRef for child nodes to indicate the absence of a child.
	//nodes []node
	//key []key[B]
	bits   []B
	len    []uint8
	offset []uint8
	left   []nodeRef
	right  []nodeRef
	entry  []bool // TODO could store this bit in offset

	// Values are indexed by the node's index in the nodes slice.
	value map[nodeRef]T
}

// newNode creates a new node in t with the provided key.
func (t *tree[B, T]) newNode(k key[B]) nodeRef {
	//t.key = append(t.key, k)
	//func (t *tree[B, T]) newNode(b B, s seg) nodeRef {
	t.bits = append(t.bits, k.content)
	t.len = append(t.len, k.len)
	t.offset = append(t.offset, k.offset)
	t.left = append(t.left, absent)
	t.right = append(t.right, absent)
	t.entry = append(t.entry, false)
	return nodeRef(len(t.bits) - 1)
}

// newTree returns a new tree.
func newTree[B KeyBits[B], T any](nodesCap int, valuesCap int) *tree[B, T] {
	t := &tree[B, T]{
		bits:   make([]B, nodesCap),
		len:    make([]uint8, nodesCap),
		offset: make([]uint8, nodesCap),
		left:   make([]nodeRef, nodesCap),
		right:  make([]nodeRef, nodesCap),
		entry:  make([]bool, nodesCap),
		value:  make(map[nodeRef]T, valuesCap),
	}
	t.newNode(key[B]{})
	return t
}

// setValue sets n's value to v and returns n.
func (t *tree[B, T]) setValue(n nodeRef, v T) nodeRef {
	t.value[n] = v
	t.entry[n] = true
	return n
}

// clearEntry removes the entry and value from t.
func (t *tree[B, T]) clearEntry(n nodeRef) {
	t.entry[n] = false
	delete(t.value, n)
}

// childAt returns the index of the child of n specified by b.
func (t *tree[B, T]) childAt(n nodeRef, b bit) nodeRef {
	if b == bitR {
		return t.right[n]
	}
	return t.left[n]
}

// childAtBool returns the index of the child of n specified by b.
func (t *tree[B, T]) childAtBool(n nodeRef, b bool) nodeRef {
	if b {
		return t.right[n]
	}
	return t.left[n]
}

func childAtBool(left, right []nodeRef, n nodeRef, b bool) nodeRef {
	if b {
		return right[n]
	}
	return left[n]
}

// setChildAt assigns o as the child of n specified by b.
func (t *tree[B, T]) setChildAt(b bit, n, o nodeRef) {
	if b == bitR {
		t.right[n] = o
		return
	}
	t.left[n] = o
}

// setChild sets o as the appropriate child of n, choosing which child position
// based on the bit at the beginning of o's key segment (i.e. at key.offset).
//
// If o == nodeRef(0), setChild does nothing.
func (t *tree[B, T]) setChild(n, o nodeRef) nodeRef {
	if o == absent {
		return o
	}
	//oKey := t.key[o]
	//b := oKey.Bit(oKey.seg.offset)
	t.setChildAt(t.bits[o].Bit(t.offset[o]), n, o)
	//t.setChildAt(b, n, o)
	return n
}

// TODO document
func (t *tree[B, T]) Cursor() treeCursor[B, T] {
	return treeCursor[B, T]{t, absent}
}

// String returns a string representation of t, showing its structure and
// values.
func (t *tree[B, T]) String() string {
	return t.Cursor().stringImpl("", "", false)
}

// Copy returns a copy of t.
//
// Note: values are copied using regular assignment, so if the values are
// pointers, the copied tree will share references with the original.
//
// Note: this has the side effect of garbage collecting the nodes slice and
// values map (for the copy, not the original).
func (t *tree[B, T]) Copy() *tree[B, T] {
	return t.Cursor().Copy().tree
}

// treeCursor is used for recursive methods that operate on a tree.
// It includes a nodeRef to track the current position in the tree.
type treeCursor[B KeyBits[B], T any] struct {
	*tree[B, T]
	node nodeRef
}

// tc2 is used for traversing two trees simultaneously.
type tc2[B KeyBits[B], T any] [2]treeCursor[B, T]

// Key returns the key of the current node.
func (t treeCursor[B, T]) Key() key[B] {
	return key[B]{t.len[t.node], t.offset[t.node], t.bits[t.node]}
}

func (t treeCursor[B, T]) HasEntry() bool {
	return t.entry[t.node]
}

func (t treeCursor[B, T]) ClearEntry() treeCursor[B, T] {
	t.tree.clearEntry(t.node)
	return t
}

func (t treeCursor[B, T]) Value() (T, bool) {
	var empty T
	if !t.HasEntry() {
		return empty, false
	}
	val, ok := t.value[t.node]
	return val, ok
}

// ChildAt returns a cursor positioned at the child (selected by b) of the
// current node. If the child doesn't exist, ChildAt returns (t, false).
func (t treeCursor[B, T]) ChildAt(b bit) (treeCursor[B, T], bool) {
	child := t.tree.childAt(t.node, b)
	return treeCursor[B, T]{t.tree, child}, child != absent
}

// ChildAtBool returns a cursor positioned at the child (selected by b) of the
// current node. If the child doesn't exist, ChildAt returns (t, false).
func (t treeCursor[B, T]) ChildAtBool(b bool) (treeCursor[B, T], bool) {
	child := t.tree.childAtBool(t.node, b)
	return treeCursor[B, T]{t.tree, child}, child != absent
}

// NewChild adds a new node with key k as a child of the current node and
// returns a cursor positioned at the new node.
func (t treeCursor[B, T]) NewChild(k key[B]) treeCursor[B, T] {
	child := t.newNode(k)
	t.SetChild(child)
	return treeCursor[B, T]{t.tree, child}
}

// NewChildAt creates a zero-valued node as the child (selected by b) of the
// current node.
func (t treeCursor[B, T]) NewChildAt(b bit) treeCursor[B, T] {
	var k key[B]
	child := t.newNode(k)
	t.SetChildAt(b, child)
	return treeCursor[B, T]{t.tree, child}
}

// NewParent creates a new node having k as its key and the current node as its
// sole child, and returns a cursor pointing to this new node.
func (t treeCursor[B, T]) NewParent(k key[B]) treeCursor[B, T] {
	t.SetOffset(k.len)
	parent := t.newNode(k)
	t.tree.setChild(parent, t.node)
	return treeCursor[B, T]{t.tree, parent}
}

// SetNode replaces the node referred to by the cursor with the provided one.
//func (t treeCursor[B, T]) SetNode(n node) treeCursor[B, T] {
//	t.nodes[t.node] = n
//	return t
//}

func emptyBits[B KeyBits[B]]() B {
	var b B
	return b
}

// DeleteNode removes the current node from the tree and returns a reference to
// the node that replaces it.
func (t treeCursor[B, T]) DeleteNode() {
	//t.key[t.node] = key[B]{}
	t.bits[t.node] = emptyBits[B]()
	t.len[t.node] = 0
	t.offset[t.node] = 0
	t.left[t.node] = absent
	t.right[t.node] = absent
	t.entry[t.node] = false
	delete(t.value, t.node)
}

// SetChild sets o as the appropriate child of the current node.
func (t treeCursor[B, T]) SetChild(o nodeRef) {
	t.tree.setChild(t.node, o)
}

// SetChildAt sets o as the child of the current node selected by b.
func (t treeCursor[B, T]) SetChildAt(b bit, o nodeRef) treeCursor[B, T] {
	t.setChildAt(b, t.node, o)
	return t
}

// SetValue updates the value of the current node.
func (t treeCursor[B, T]) SetValue(v T) treeCursor[B, T] {
	t.tree.setValue(t.node, v)
	return t
}

// SetValueFrom sets the current node's value to o's value if it exists.
func (t treeCursor[B, T]) SetValueFrom(o treeCursor[B, T]) treeCursor[B, T] {
	if v, ok := o.Value(); ok {
		t.SetValue(v)
	}
	return t
}

// SetOffset sets the current node's offset to the provided value.
func (t treeCursor[B, T]) SetOffset(offset uint8) treeCursor[B, T] {
	t.offset[t.node] = offset
	return t
}

// TODO: get rid of hideVal if possible
// TODO: if hideVal, still distinguish entries from non-entries
func (t treeCursor[B, T]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	nk := t.Key()
	if hideVal || !t.HasEntry() {
		ret = fmt.Sprintf("%s%s%s\n", indent, pre, nk.StringRel())
	} else {
		ret = fmt.Sprintf("%s%s%s: %v\n", indent, pre, nk.StringRel(),
			t.tree.value[t.node])
	}
	if left, ok := t.ChildAt(bitL); ok {
		ret += left.stringImpl(indent+"  ", "L:", hideVal)
	}
	if right, ok := t.ChildAt(bitR); ok {
		ret += right.stringImpl(indent+"  ", "R:", hideVal)
	}
	return ret
}

// Size returns the number of entries in t.
// TODO: keep track of this instead of calculating it lazily
func (t treeCursor[B, T]) Size() (size int) {
	if t.HasEntry() {
		size += 1
	}
	if left, ok := t.ChildAt(bitL); ok {
		size += left.Size()
	}
	if right, ok := t.ChildAt(bitR); ok {
		size += right.Size()
	}
	return
}

// Copy returns a copy of t.
//
// Note: values are copied using regular assignment, so if the values are
// pointers, the copied tree will share references with the original.
//
// Note: this has the side effect of "garbage collecting" the nodes slice and
// values map (for the copy, not the original): unreachable nodes and their
// values are not copied.
func (t treeCursor[B, T]) Copy() treeCursor[B, T] {
	return newTree[B, T](len(t.bits), len(t.value)).Cursor().CopyFrom(t)
}

// CopyFrom copies o to t.
// TODO: is it ok if this overwrites? yes.
func (t treeCursor[B, T]) CopyFrom(o treeCursor[B, T]) treeCursor[B, T] {
	s := stack[tc2[B, T]]{}
	s.Push(tc2[B, T]{o, t})
	for !s.IsEmpty() {
		c := s.Pop()
		src, dst := c[0], c[1]
		//srcN := src.Node()
		//dst.SetNode(node{key: srcN.key, hasEntry: srcN.hasEntry})
		//dst.SetNode(node{key: src.Key()})
		//dst.key[dst.node] = src.key[src.node]
		dst.bits[dst.node] = src.bits[src.node]
		dst.len[dst.node] = src.len[src.node]
		dst.offset[dst.node] = src.offset[src.node]
		dst.SetValueFrom(src)
		for _, bit := range eachBit {
			if srcChild, srcOk := src.ChildAt(bit); srcOk {
				s.Push(tc2[B, T]{srcChild, dst.NewChildAt(bit)})
			}
		}
	}
	return t
}

// Insert inserts value v at key k with path compression and moves the cursor
// to the inserted node.
// TODO: Perhaps Insert and other recursive methods should just return nodeRefs.
// It's very common to access the node field immediately.
// Alternatively, maybe SetChild should accept a treeCursor or there should be
// a SetChildFrom, but that seems silly if we don't often use the cursor
// returned by these methods.
func (t treeCursor[B, T]) Insert(k key[B], v T) treeCursor[B, T] {
	tKey := t.Key()
	// Inserting at the current node itself
	if tKey.EqualFromRoot(k) {
		return t.SetValue(v)
	}

	com := tKey.CommonPrefixLen(k)
	switch {
	// Inserting at a descendant; recurse into the appropriate child
	case com == tKey.len:
		child, ok := t.ChildAt(k.Bit(tKey.len))
		if !ok {
			child = t.NewChild(k.Rest(tKey.len)).SetValue(v)
		}
		child = child.Insert(k, v)
		t.SetChild(child.node)
		return t
	// Inserting at a prefix of tKey; create a new parent node with t as its
	// sole child
	case com == k.len:
		return t.NewParent(k).SetValue(v)
	// Neither is a prefix of the other; create a new parent at their common
	// prefix with children t and its new sibling
	default:
		parent := t.NewParent(tKey.Truncated(com))
		parent.NewChild(k.Rest(com)).SetValue(v)
		return parent
	}
}

// Remove removes the exact provided key from the tree, if it exists, with
// path compression, and returns a reference to the node that replaces t's
// current node, if any.
func (t treeCursor[B, T]) Remove(k key[B]) nodeRef {
	tKey := t.Key()
	switch {
	// Removing t itself
	case k.EqualFromRoot(tKey):
		if t.HasEntry() {
			t.ClearEntry()
		}
		left, leftOk := t.ChildAt(bitL)
		right, rightOk := t.ChildAt(bitR)
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
	case tKey.IsPrefixOf(k):
		bit := k.Bit(tKey.len)
		if child, ok := t.ChildAt(bit); ok {
			// We need to use SetChildAt because the returned nodeRef may be 0
			t.SetChildAt(bit, child.Remove(k))
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
func (t treeCursor[B, T]) SubtractKey(k key[B]) nodeRef {
	tKey := t.Key()
	// This whole branch is being subtracted; no need to traverse further
	if tKey.EqualFromRoot(k) || k.IsPrefixOf(tKey) {
		return 0
	}
	// A child of t is being subtracted
	if tKey.IsPrefixOf(k) {
		bit := k.Bit(tKey.len)
		child, ok := t.ChildAt(bit)
		if ok {
			// We need to use SetChildAt because the returned nodeRef may be 0
			t.SetChildAt(bit, child.SubtractKey(k.Rest(tKey.len)))
		} else {
			// sr: I'm not sure if this is right. It was:
			// t.insertHole(k, t.value)
			if val, ok := t.Value(); ok {
				t.insertHole(k, val)
			}
		}
		// TODO: is this just IsEmpty?
		if t.right[t.node] == 0 && t.left[t.node] == 0 && !t.HasEntry() {
			return 0
		}
	}
	return t.node
}

// SubtractTree removes all entries from t that have counterparts in o. If a
// child of t is removed, then new nodes may be created to fill in the gaps
// around the removed node.
//
// TODO: this method only makes sense in the context of a PrefixSet.
// "subtracting" a whole key-value entry from another isn't meaningful. So
// maybe we need two types of trees: value-bearing ones, and others that just
// have value-less entries.
func (t treeCursor[B, T]) SubtractTree(o treeCursor[B, T]) nodeRef {
	tKey, oKey := t.Key(), o.Key()

	//if o.Node().hasEntry {
	if o.HasEntry() {
		// This whole branch is being subtracted; no need to traverse further
		if oKey.IsPrefixOf(tKey) {
			return 0
		}
		// A descendant of t is being subtracted
		if tKey.IsPrefixOf(oKey) {
			// sr: I'm not sure if this is right. It was:
			// t.insertHole(o.key, t.value)
			if val, ok := t.Value(); ok {
				t.insertHole(oKey, val)
			}
		}
	}
	// Consider the children of both t and o
	for _, bit := range eachBit {
		tChild, _ := t.ChildAt(bit)
		oChild, oOk := o.ChildAt(bit)
		if oOk {
			// We need to use SetChildAt because the returned nodeRef may be 0
			t.SetChildAt(bit, tChild.SubtractTree(oChild))
		}

		// TODO remove; keeping as reference for now
		//if *oChild != nil {
		//	if *tChild == nil {
		//		tChild = &t
		//	}
		//	*tChild = (*tChild).subtractTree(*oChild)
		//}
	}
	return t.node
}

// IsEmpty reports whether the tree is empty.
func (t treeCursor[B, T]) IsEmpty() bool {
	//n := t.Node()
	//return n.key.IsZero() && n.left == 0 && n.right == 0
	return t.Key().IsZero() && t.left[t.node] == absent && t.right[t.node] == absent
}

// MergeTree modifies t so that it is the union of the entries of t and o.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
func (t treeCursor[B, T]) MergeTree(o treeCursor[B, T]) treeCursor[B, T] {
	// If o is empty, then the union is just t
	if o.IsEmpty() {
		return t
	}
	tKey, oKey := t.Key(), o.Key()

	if tKey.EqualFromRoot(oKey) {
		if !t.HasEntry() {
			t.SetValueFrom(o)
		}

		for _, bit := range eachBit {
			if oChild, oOk := o.ChildAt(bit); oOk {
				tChild, _ := t.ChildAt(bit)
				// sr: this assignment was unconditional
				t.SetChild(tChild.MergeTree(oChild).node)
			}
		}
		return t
	}

	com := tKey.CommonPrefixLen(oKey)

	switch {
	// tKey is a prefix of oKey
	case com == tKey.len:
		// Traverse t in the direction of o
		if tChild, ok := t.ChildAt(oKey.Bit(tKey.len)); ok {
			// sr: the condition is explicit here (ok)
			t.SetChild(tChild.MergeTree(o).node)
		} else {
			tChild.CopyFrom(o).SetOffset(tKey.len)
		}
		return t
	// o.key is a prefix of tKey
	case com == oKey.len:
		// o needs to inserted as a parent of t regardless of whether o has an
		// entry (if the node exists in the o tree, it will need to be in the
		// union tree). Insert it and continue traversing from there.
		return t.NewParent(oKey).SetValueFrom(o).MergeTree(o)
	// Neither is a prefix of the other
	default:
		// Insert a new parent above t, and create a new sibling for t having
		// o's key and value.
		parent := t.NewParent(tKey.Truncated(com))
		parent.NewChild(oKey.Rest(com)).SetValueFrom(o)
		return parent
	}
}

func (t treeCursor[B, T]) intersectTreeImpl(
	o treeCursor[B, T],
	tPathHasEntry, oPathHasEntry bool,
) treeCursor[B, T] {

	// If o is an empty tree, then any intersection with it is also empty
	if o.IsEmpty() {
		t.DeleteNode()
		return t
	}

	tKey, oKey := t.Key(), o.Key()

	if tKey.EqualFromRoot(oKey) {
		// Consider t and o themselves.
		// If there is no entry in o at t.key or above it, then remove t's
		// entry.
		//
		// TODO should this be t.remove(t.key)? Could we end up with an
		// unnecessary prefix node?
		if t.HasEntry() && !(o.HasEntry() || oPathHasEntry) {
			t.ClearEntry()
			// We need to remember that t had an entry here so that o's
			// descendants are kept
			tPathHasEntry = true
		}

		// Consider the children of t and o
		for _, bit := range eachBit {
			tChild, tOk := t.ChildAt(bit)
			oChild, oOk := o.ChildAt(bit)
			switch {
			case !tOk && oOk && (t.HasEntry() || tPathHasEntry):
				tChild.CopyFrom(oChild) //*tChild = (*oChild).copy()
			case tOk && !oOk && !(o.HasEntry() || oPathHasEntry):
				tChild.DeleteNode() //*tChild = nil
			case tOk && oOk:
				// sr: the condition is explicit here (tOk)
				t.SetChild(tChild.intersectTreeImpl(
					oChild,
					t.HasEntry() || tPathHasEntry,
					o.HasEntry() || oPathHasEntry,
				).node)
			}
		}
		return t
	}

	com := tKey.CommonPrefixLen(oKey)

	switch {
	// t.key is a prefix of o.key
	case com == tKey.len:
		if t.HasEntry() {
			// o is more specific than t. If o has no entry above it, then t
			// itself is not in the intersection...
			if !oPathHasEntry {
				t.ClearEntry()
			}
			// ...but o is, because it's under t which has an entry.
			if val, ok := o.Value(); ok {
				t.Insert(oKey, val)
			}
		}

		// t forks in the middle of oKey. To take the intersection, we
		// need to traverse t toward oKey and prune the other child of t.
		//
		// The bit of oKey just after the common prefix determines which
		// of t's children to follow and which to remove.
		// e.g. t=00, o=000 -> follow left, remove right
		// sr:
		// - this could be natural to return uint64's
		bit := oKey.Bit(com)
		tFollow, tFollowOk := t.ChildAt(bit)
		tRemove, _ := t.ChildAt(inv(bit))

		// Traverse t in the direction of oKey.
		if tFollowOk {
			// sr: the condition is explicit here (tFollowOk)
			t.SetChild(tFollow.intersectTreeImpl(o,
				t.HasEntry() || tPathHasEntry,
				o.HasEntry() || oPathHasEntry,
			).node)
		}

		// Remove the child of t that diverges from o.
		//
		// Exception: if o has an ancestor entry, then we don't need to remove
		// anything under t. TODO: is this check necessary?
		if !oPathHasEntry {
			tRemove.DeleteNode()
		}

	// o.key is a prefix of t.key
	case com == oKey.len:
		// o forks in the middle of t.key. Similar to above.
		oFollow, oOk := o.ChildAt(tKey.Bit(com))

		// Traverse o in the direction of t.key.
		//
		// We don't need to visit t's children here; if there is intersection
		// under t, it will be handled within the call below by one of the
		// above cases.
		if oOk {
			// sr: is this right? it was:
			//t = t.intersectTreeImpl(*oChildFollow,
			//	t.hasEntry || tPathHasEntry,
			//	o.hasEntry || oPathHasEntry,
			//)
			t.SetChild(t.intersectTreeImpl(oFollow,
				t.HasEntry() || tPathHasEntry,
				o.HasEntry() || oPathHasEntry,
			).node)
		}
	// Neither is a prefix of the other, so the intersection is empty
	default:
		t.DeleteNode()
	}

	return t
}

// intersectTree modifies t so that it is the intersection of the entries of t
// and o: an entry is included iff it (1) is present in both trees or (2) is
// present in one tree and has a parent entry in the other tree.
//
// TODO: same problem as subtractTree; only makes sense for PrefixSets.
func (t treeCursor[B, T]) IntersectTree(o treeCursor[B, T]) treeCursor[B, T] {
	return t.intersectTreeImpl(o, false, false)
}

// insertHole removes k and sets t, and all of its descendants, to v.
func (t treeCursor[B, T]) insertHole(k key[B], v T) nodeRef {
	tKey := t.Key()
	switch {
	// Removing t itself (no descendants will receive v)
	case t.Key().EqualFromRoot(k):
		return 0
	// k is a descendant of t; start digging a hole to k
	case t.Key().IsPrefixOf(k):
		t.ClearEntry()

		// Create a new sibling to receive v if needed, then continue traversing
		bit := k.Bit(tKey.len)
		child, _ := t.ChildAt(bit)
		_, siblingOk := t.ChildAt(inv(bit))
		if !siblingOk {
			//*sibling = newTree[B, T](t.key.next((^bit) & 1)).setValue(v)
			t.NewChild(tKey.Next(inv(bit))).SetValue(v)
		}
		//*child = newTree[B, T](t.key.next(bit)).insertHole(k, v)
		t.SetChild(child.insertHole(k, v))

		// Create a new sibling to receive v if needed, then continue traversing
		//bit := k.bit(tKey.seg.len)
		//child, sibling := t.children(bit)
		//if *sibling == nil {
		//	*sibling = newNode[B, T](tKey.next(inv(bit))).setValue(v)
		//}
		//*child = newNode[B, T](tKey.next(bit)).insertHole(k, v)
		return t.node

	// Nothing to do
	default:
		return t.node
	}
}

// walk traverses the tree starting at this tree's root, following the
// provided path and calling fn(node) at each visited node.
//
// When the provided path is exhausted, walk continues by visiting all
// children.
//
// If fn returns true, then walk stops traversing any deeper.
func (t treeCursor[B, T]) walk(path key[B], fn func(treeCursor[B, T]) bool) {
	// Follow provided path directly until it's exhausted
	var ok bool
	for ok = true; ok && t.Key().len < path.len; t, ok = t.pathNext(path) {
		if !t.Key().IsZero() {
			if fn(t) {
				return
			}
		}
	}

	if t.IsEmpty() {
		return
	}

	// After path is exhausted, visit all children
	var st stack[treeCursor[B, T]]
	var stop bool
	st.Push(t)
	for !st.IsEmpty() {
		stop = false
		t = st.Pop()
		if !t.Key().IsZero() {
			stop = fn(t)
		}
		// TODO 32
		if t.Key().len < 128 && !stop {
			if right, ok := t.ChildAt(bitR); ok {
				st.Push(right)
			}
			if left, ok := t.ChildAt(bitL); ok {
				st.Push(left)
			}
		}
	}
}

// pathNext returns a cursor pointing to the child of t which is next in the
// traversal of the specified path.
func (t treeCursor[B, T]) pathNext(path key[B]) (treeCursor[B, T], bool) {
	return t.ChildAt(path.Bit(t.Key().len))
}

// get returns the value associated with the exact key provided, if it exists.
// TODO this is only applicable to value-bearing trees
func (t treeCursor[B, T]) Get(k key[B]) (val T, ok bool) {
	for t, pathOk := t.pathNext(k); pathOk; t, pathOk = t.pathNext(k) {
		tKey := t.Key()
		if !tKey.IsZero() && tKey.len >= k.len {
			if tKey.EqualFromRoot(k) && t.HasEntry() {
				val, ok = t.Value()
			}
			break
		}
	}
	return
}

// contains returns true if this tree includes the exact key provided.
func (t treeCursor[B, T]) Contains(k key[B]) (ret bool) {
	for t, pathOk := t.pathNext(k); pathOk; t, pathOk = t.pathNext(k) {
		tKey := t.Key()
		if !tKey.IsZero() {
			if ret = (tKey.EqualFromRoot(k) && t.HasEntry()); ret {
				break
			}
		}
	}
	return
}

func (t *tree[B, T]) equalPrefix(n nodeRef, k key[uint128]) bool {
	return t.bits[n].EqualPrefix(k.content, t.len[n])
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
// TODO strict
func (t treeCursor[B, T]) Encompasses(k key[uint128]) (ret bool) {
	n := childAtBool(t.left, t.right, t.node, k.content.BitBool(t.tree.len[t.node]))
	for n != absent {
		if ret = t.tree.entry[n] && t.tree.len[n] <= k.len && t.tree.equalPrefix(n, k); ret {
			break
		}
		n = childAtBool(t.left, t.right, n, k.content.BitBool(t.tree.len[n]))
	}
	return
}

// encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
// TODO strict
func (t treeCursor[B, T]) EncompassesStrict(k key[uint128]) (ret bool) {
	b128 := k.content.To128()
	//n := t.tree.childAt(t.node, k128.Bit(t.tree.key[t.node].seg.len))
	n := childAtBool(t.left, t.right, t.node, b128.BitBool(t.tree.len[t.node]))
	for n != absent {
		if ret = t.tree.entry[n] && t.tree.len[n] < k.len && t.tree.equalPrefix(n, k); ret {
			break
		}
		//n = t.tree.childAt(n, k128.Bit(t.tree.key[n].seg.len))
		n = childAtBool(t.left, t.right, n, b128.BitBool(t.tree.len[n]))
	}
	return
}

// rootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
// TODO strict
func (t treeCursor[B, T]) RootOf(k key[B], _ bool) (outKey key[B], val T, ok bool) {
	t.walk(k, func(n treeCursor[B, T]) bool {
		if n.Key().IsPrefixOf(k) && n.HasEntry() {
			outKey = n.Key()
			val, ok = n.Value()
			return true
		}
		return false
	})
	return
}

// parentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t treeCursor[B, T]) ParentOf(k key[B], strict bool) (outKey key[B], val T, ok bool) {
	t.walk(k, func(n treeCursor[B, T]) bool {
		if n.Key().IsPrefixOf(k) && n.HasEntry() {
			outKey = n.Key()
			val, ok = n.Value()
		}
		return false
	})
	return
}

// descendantsOf returns the sub-tree containing all descendants of the
// provided key. The key itself will be included if it has an entry in the
// tree, unless strict == true. descendantsOf returns an empty tree if the
// provided key is not in the tree.
func (t treeCursor[B, T]) DescendantsOf(k key[B], strict bool) (ret treeCursor[B, T]) {
	t.walk(k, func(n treeCursor[B, T]) bool {
		if k.IsPrefixOf(n.Key()) {
			ret = n.Copy()
			ret.SetOffset(0)
			if !(strict && n.Key().EqualFromRoot(k)) {
				ret.SetValueFrom(n)
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
func (t treeCursor[B, T]) AncestorsOf(k key[B], strict bool) (ret treeCursor[B, T]) {
	ret = newTree[B, T](1, 1).Cursor()
	t.walk(k, func(n treeCursor[B, T]) bool {
		if !n.Key().IsPrefixOf(k) {
			return true
		}
		if n.HasEntry() && !(strict && n.Key().EqualFromRoot(k)) {
			// TODO always expect ok == true if hasEntry == true
			if val, ok := n.Value(); ok {
				ret.Insert(n.Key(), val)
			}
		}
		return false
	})
	return
}

// filter updates t to include only the keys encompassed by o.
//
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
func (t treeCursor[B, T]) Filter(o treeCursor[B, bool]) {
	remove := make([]key[B], 0)
	var k key[B]
	t.walk(k, func(n treeCursor[B, T]) bool {
		if !o.Encompasses(n.Key().To128()) { // TODO
			remove = append(remove, n.Key())
		}
		return false
	})
	for _, k := range remove {
		t.Remove(k)
	}
}

// filterCopy returns a recursive copy of t that includes only keys that are
// encompassed by o.
// TODO: I think this can be done more efficiently by walking t and o
// at the same time.
// TODO: does it make sense to have both this method and filter()?
func (t treeCursor[B, T]) FilterCopy(o treeCursor[B, bool]) treeCursor[B, T] {
	ret := newTree[B, T](1, 1).Cursor()
	var k key[B]
	t.walk(k, func(n treeCursor[B, T]) bool {
		if n.HasEntry() && o.Encompasses(n.Key().To128()) { // TODO
			// TODO always expect ok == true if hasEntry == true
			if val, ok := n.Value(); ok {
				ret = ret.Insert(n.Key(), val)
			}
		}
		return false
	})
	return ret
}

// overlapsKey reports whether any key in t overlaps k.
func (t treeCursor[B, T]) OverlapsKey(k key[B]) (ret bool) {
	t.walk(k, func(n treeCursor[B, T]) bool {
		if !n.HasEntry() {
			return false
		}
		if n.Key().IsPrefixOf(k) || k.IsPrefixOf(n.Key()) {
			ret = true
			return true
		}
		return false
	})
	return
}
