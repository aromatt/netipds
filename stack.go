package netipds

// The tree can have a depth of 129 because we support storing a node with
// length 0, e.g., 0.0.0.0/0. If this prefix is in the tree, then it and its
// children will both have offset 0. This is the only node that may have the
// same offset as its children. TODO but this probably breaks traversal somewhere!
const stackMaxDepth = 129

// stack is used for depth-first traversals without recursion or heap
// allocation.
type stack[T any] struct {
	data [stackMaxDepth]T
	// top starts at 0, so it is the index of the next available slot.
	top int
}

// Push adds an element to the top of the stack. Panics if stack is full.
func (s *stack[T]) Push(value T) {
	s.data[s.top] = value
	s.top++
}

// IsEmpty reports whether the stack is empty.
func (s *stack[T]) IsEmpty() bool {
	return s.top <= 0
}

// Pop removes and returns the element at the top of the stack. Panics if stack
// is empty (use IsEmpty()).
func (s *stack[T]) Pop() T {
	s.top--
	value := s.data[s.top]
	return value
}
