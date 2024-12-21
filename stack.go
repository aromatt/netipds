package netipds

// stack is used for depth-first traversals without recursion or heap
// allocation.
type stack[T any] struct {
	data [128]T
	// top starts at 0, so it is the index of the next available slot.
	top int
}

// Push adds an element to the top of the stack. Fails and returns false if
// stack is full.
func (s *stack[T]) Push(value T) bool {
	if s.top >= len(s.data) {
		return false
	}
	s.data[s.top] = value
	s.top++
	return true
}

// Pop removes and returns the element at the top of the stack. Fails and
// returns false if stack is empty.
func (s *stack[T]) Pop() (T, bool) {
	if s.top <= 0 {
		var empty T
		return empty, false
	}
	s.top--
	value := s.data[s.top]
	return value, true
}
