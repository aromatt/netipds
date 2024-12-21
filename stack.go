package netipds

type stack[T any] struct {
	data [128]T
	top  int
}

// Push adds an element to the top of the stack.
func (s *stack[T]) Push(value T) {
	if s.top >= len(s.data)-1 {
		panic("stack overflow") // Prevent pushing beyond fixed capacity
	}
	s.top++
	s.data[s.top] = value
}

// Pop removes and returns the element from the top of the stack.
func (s *stack[T]) Pop() T {
	if s.top < 0 {
		panic("stack underflow") // Prevent popping when empty
		//var zero T
		//return zero
	}
	value := s.data[s.top]
	s.top--
	return value
}
