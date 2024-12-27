package netipds

// stack is used for depth-first traversals without recursion or heap
// allocation.
type stack[T any] struct {
	data [128]T
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
