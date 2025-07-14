package netipds

// The max depth *would* be 128 to match the maximum length of a key in our
// tree, but it is 129 because there is always a root node with a length of
// zero.
//
// The root node may contain an entry: we use it to support the special
// prefixes 0.0.0.0/0 and ::/0. Since the root node has a length of zero, it
// does not own any of the available 128 bits, therefore the tree can have a
// depth of 129.
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
