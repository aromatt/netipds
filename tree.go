package netipmap

import (
	"fmt"
)

type Tree[T any] interface {
	IsEmpty() bool
	PrettyPrint(string, string)
	Set(key, T) Tree[T]
	SetKey(key) Tree[T]
	Remove(key) Tree[T]
	Walk(key, func(Tree[T]) bool)
	Get(key) (T, bool)
	Contains(key) bool
	Encompasses(key, bool) bool
	Covers(key, bool) bool
	RootOf(key, bool) (key, T, bool)
	ParentOf(key, bool) (key, T, bool)
	DescendantsOf(key, bool) Tree[T]
	AncestorsOf(key, bool) Tree[T]
	Filter(Tree[T])
	Copy() Tree[T]
	Key() key
	// TODO we can probably get rid of these and do ChildAt(0) and ChildAt(1)
	Left() Tree[T]
	Right() Tree[T]
	Value() T
	MergeChild(Tree[T]) Tree[T]
}

// zv returns the zero value of type T.
func zv[T any]() T { var v T; return v }

// emptyTree is an empty tree.
type emptyTree[T any] struct{}

func (emptyTree[T]) IsEmpty() bool                     { return true }
func (emptyTree[T]) PrettyPrint(string, string)        {}
func (emptyTree[T]) Remove(key) Tree[T]                { return nil }
func (emptyTree[T]) Walk(key, func(Tree[T]) bool)      {}
func (emptyTree[T]) Get(key) (T, bool)                 { return zv[T](), false }
func (emptyTree[T]) Contains(key) bool                 { return false }
func (emptyTree[T]) Encompasses(key, bool) bool        { return false }
func (emptyTree[T]) Covers(key, bool) bool             { return false }
func (emptyTree[T]) RootOf(key, bool) (key, T, bool)   { return key{}, zv[T](), false }
func (emptyTree[T]) ParentOf(key, bool) (key, T, bool) { return key{}, zv[T](), false }
func (t emptyTree[T]) DescendantsOf(key, bool) Tree[T] { return t }
func (t emptyTree[T]) AncestorsOf(key, bool) Tree[T]   { return t }
func (emptyTree[T]) Filter(Tree[T])                    { return }
func (t emptyTree[T]) Copy() Tree[T]                   { return t }
func (emptyTree[T]) Key() key                          { return key{} }
func (emptyTree[T]) Left() Tree[T]                     { return nil }
func (emptyTree[T]) Right() Tree[T]                    { return nil }
func (emptyTree[T]) Value() T                          { return zv[T]() }

func (t emptyTree[T]) Set(k key, v T) Tree[T] {
	return &tree[T]{key: k, value: v, hasValue: true}
}

func (t emptyTree[T]) SetKey(k key) Tree[T] {
	return &tree[T]{key: k}
}

func (t emptyTree[T]) MergeChild(o Tree[T]) Tree[T] {
	return o
}

// tree is a binary radix tree with path compression.
type tree[T any] struct {
	key   key
	value T
	left  Tree[T]
	right Tree[T]

	// Not every node has a value. A node may be just a shared prefix.
	hasValue bool
}

// newTree returns a new tree with the provided key.
func newTree[T any](k key) *tree[T] {
	return &tree[T]{key: k}
}

// clearValue removes the value from t.
func (t *tree[T]) clearValue() {
	t.value = zv[T]()
	t.hasValue = false
}

// setValue sets t's value to v and returns t.
func (t *tree[T]) setValue(v T) *tree[T] {
	t.value = v
	t.hasValue = true
	return t
}

// setValueFrom sets t's value to m's value and returns t.
func (t *tree[T]) setValueFrom(o *tree[T]) *tree[T] {
	if o != nil && o.hasValue {
		return t.setValue(o.value)
	}
	return t
}

// moveValueFrom moves m's value to t (removing it from m) and returns t.
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
func (t *tree[T]) setChildren(left Tree[T], right Tree[T]) *tree[T] {
	t.left = left
	t.right = right
	return t
}

// setChildrenFrom sets t's children to o's (without copying them) and returns t.
func (t *tree[T]) setChildrenFrom(o Tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	t = t.setChildren(o.Left(), o.Right())
	return t
}

// copyChildrenFrom sets t's children to copies of m's children and returns t.
func (t *tree[T]) copyChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	return t.setChildren(o.left.Copy(), o.right.Copy())
}

// moveChildrenFrom moves m's children to t (removing them from m) and returns t.
func (t *tree[T]) moveChildrenFrom(o *tree[T]) *tree[T] {
	if o == nil {
		return t
	}
	t.setChildrenFrom(o)
	o.setChildren(nil, nil)
	return t
}

func (t *tree[T]) IsEmpty() bool  { return false }
func (t *tree[T]) Key() key       { return t.key }
func (t *tree[T]) Left() Tree[T]  { return t.left }
func (t *tree[T]) Right() Tree[T] { return t.right }
func (t *tree[T]) Value() T       { return t.value }

// SetKey sets t's key to k and returns t.
func (t *tree[T]) SetKey(k key) Tree[T] {
	t.key = k
	return t
}

func (t *tree[T]) MergeChild(right bool) Tree[T] {
	if right && t.right != nil {
		return t.right
	}
	k := o.Key()
	k.offset = t.key.offset
	o.SetKey(k)
	return o
}

// Copy returns a copy of t, creating copies of all descendants of t in the
// process. If t is nil, copy returns nil.
func (t *tree[T]) Copy() Tree[T] {
	if t == nil {
		return nil
	}
	return newTree[T](t.key).copyChildrenFrom(t).setValueFrom(t)
}

// PrettyPrint prints the tree in a human-readable format.
func (t *tree[T]) PrettyPrint(indent string, prefix string) {
	fmt.Printf("%s%s%s: %v\n", indent, prefix, t.key, t.value)
	if t.left != nil {
		t.left.PrettyPrint(indent+"  ", "L:")
	}
	if t.right != nil {
		t.right.PrettyPrint(indent+"  ", "R:")
	}
}

// set inserts the provided key and value into the tree.
// TODO revisit this
func (t *tree[T]) Set(k key, value T) Tree[T] {
	if t.key == k {
		t.value = value
		t.hasValue = true
		return t
	}

	if t.key.isPrefixOf(k) {
		// t.key is a prefix of the new key, so recurse into the
		// appropriate child of n (or create it).
		var next *Tree[T]
		// t.key.len < l.len because t.key is strictly a prefix of l
		if zero, _ := k.isBitZero(t.key.len); zero {
			next = &t.left
		} else {
			next = &t.right
		}
		if *next == nil {
			*next = newTree[T](k.rest(t.key.len)).setValue(value)
		} else {
			(*next).Set(k.rest(t.key.len), value)
		}
	} else {
		common := t.key.commonPrefixLen(k)

		// Split t and create two new children: an "heir" to inherit t's
		// suffix, and a sibling to handle the new suffix.
		heir := newTree[T](t.key.rest(common)).moveValueFrom(t).moveChildrenFrom(t)
		sibling := newTree[T](k.rest(common)).setValue(value)

		// The bit after the common prefix determines which child will handle
		// which suffix.
		// TODO check ok
		if zero, _ := t.key.isBitZero(common); zero {
			t.left = heir
			t.right = sibling
		} else {
			t.left = sibling
			t.right = heir
		}

		// t's key needs to be truncated at the split point
		t.key = t.key.truncated(common)
	}
	return t
}

// Remove removes the exact key provided from the tree, if it exists.
func (t *tree[T]) Remove(k key) Tree[T] {
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
			return t.MergeChild(t.right) //.SetKey().offset = t.key.offset
		case t.right == nil:
			//t.left.key.offset = t.key.offset
			return t.MergeChild(t.left)
		// This is a shared prefix node, so it needs to persist.
		default:
			return t
		}
	}

	// t.key is a prefix of the new key, so recurse into the
	// appropriate child of t.
	if t.key.isPrefixOf(k) {
		if zero, _ := k.isBitZero(t.key.len); zero {
			if t.left != nil {
				t.left = t.Left().Remove(k.rest(t.key.len))
			}
		} else {
			if t.right != nil {
				t.right = t.right.Remove(k.rest(t.key.len))
			}
		}
	}

	return t
}

// Walk traverses the tree starting at this tree's root, following the
// provided path and calling fn(node) at each visited node.
//
// The return value of fn is a boolean indicating whether traversal should
// stop.
//
// If path is the zero key, all descendants of this tree are visited.
func (t *tree[T]) Walk(k key, fn func(Tree[T]) bool) {
	// Never call fn on root node
	if !t.key.isZero() {
		if fn(t) {
			return
		}
	}

	nextPath := k.rest(t.key.len)
	zero, pathExhausted := k.isBitZero(t.key.commonPrefixLen(k))

	// Visit the child that matches the next bit in the path. If the path is
	// exhausted, visit both children.
	if (zero || !pathExhausted) && t.left != nil {
		t.left.Walk(nextPath, fn)
	}
	if (!zero || !pathExhausted) && t.right != nil {
		t.right.Walk(nextPath, fn)
	}
	return
}

// Get returns the value associated with the exact key provided, if it exists.
func (t *tree[T]) Get(k key) (val T, ok bool) {
	t.Walk(k, func(n Tree[T]) bool {
		if n.key.equalFromRoot(k) && n.hasValue {
			val, ok = n.value, true
			return true
		}
		return false
	})
	return
}

// Contains returns true if this tree includes the exact key provided.
func (t *tree[T]) Contains(k key) (ret bool) {
	t.Walk(k, func(n *tree[T]) bool {
		if ret = (n.key.equalFromRoot(k) && n.hasValue); ret {
			return true
		}
		return false
	})
	return
}

// Encompasses returns true if this tree includes a key which completely
// encompasses the provided key.
func (t *tree[T]) Encompasses(k key, strict bool) (ret bool) {
	t.Walk(k, func(n *tree[T]) bool {
		if ret = (n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue); ret {
			return true
		}
		return false
	})
	return
}

// Covers returns true if this tree includes a subset of keys that completely
// cover the provided key.
func (t *tree[T]) Covers(k key, strict bool) (ret bool) {
	// TODO implement
	panic("not implemented")
}

// RootOf returns the shortest-prefix ancestor of the key provided, if any.
// If strict == true, the key itself is not considered.
func (t *tree[T]) RootOf(k key, strict bool) (outPath key, val T, ok bool) {
	t.Walk(k, func(n *tree[T]) bool {
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue {
			outPath, val, ok = n.key, n.value, true
			return true
		}
		return false
	})
	return
}

// ParentOf returns the longest-prefix ancestor of the key provided, if any.
// If strict is true, the key itself is not considered.
func (t *tree[T]) ParentOf(k key, strict bool) (outPath key, val T, ok bool) {
	t.Walk(k, func(n *tree[T]) bool {
		if n.key.isPrefixOf(k) && !(strict && n.key == k) && n.hasValue {
			outPath, val, ok = n.key, n.value, true
		}
		return false
	})
	return
}

// DescendantsOf returns the sub-tree containing all descendants of the
// provided key. The key itself will be included if it has an entry in the
// tree, unless strict. DescendantsOf returns the empty tree if the provided
// key is not in the tree.
func (t *tree[T]) DescendantsOf(k key, strict bool) Tree[T] {
	ret := &tree[T]{}
	t.Walk(k, func(n *tree[T]) bool {
		if k.isPrefixOf(n.key) {
			ret = ret.setKey(n.key.rooted()).setValueFrom(n).setChildrenFrom(n)
			return true
		}
		return false
	})
	return
}

// AncestorsOf returns the sub-tree containing all ancestors of the provided
// key. The key itself will be included if it has an entry in the tree, unless
// strict. AncestorsOf returns an empty tree if key has no ancestors in the
// tree.
func (t *tree[T]) AncestorsOf(k key, strict bool) Tree[T] {
	ret := &tree[T]{}
	t.Walk(k, func(n *tree[T]) bool {
		if !n.key.isPrefixOf(k) {
			return true
		}
		if n.hasValue {
			ret.set(n.key, n.value)
		}
		return false
	})
	return ret
}

// Filter updates t to include only the keys encompassed by b.
// TODO: I think this can be done more efficiently by walking t and b
// at the same time.
func (t *tree[T]) Filter(o Tree[T]) {
	remove := make([]key, 0)
	t.Walk(key{}, func(n *tree[T]) bool {
		if !o.Encompasses(n.key, false) {
			remove = append(remove, n.key)
		}
		return false
	})
	for _, k := range remove {
		t.Remove(k)
	}
}
