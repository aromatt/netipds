package netipds

import (
// "encoding/binary"
// "math/bits"
)

type ipUint[U uint16 | uint64] struct {
	hi U
	lo U
}

// isZero reports whether u == 0.
//
// It's faster than u == (uint128{}) because the compiler (as of Go
// 1.15/1.16b1) doesn't do this trick and instead inserts a branch in
// its eq alg's generated code.
func (u ipUint[U]) isZero() bool { return u.hi|u.lo == 0 }

// and returns the bitwise AND of u and m (u&m).
func (u ipUint[U]) and(m ipUint[U]) ipUint[U] {
	return ipUint[U]{u.hi & m.hi, u.lo & m.lo}
}

// or returns the bitwise OR of u and m (u|m).
func (u ipUint[U]) or(m ipUint[U]) ipUint[U] {
	return ipUint[U]{u.hi | m.hi, u.lo | m.lo}
}

// not returns the bitwise NOT of u.
func (u ipUint[U]) not() ipUint[U] {
	return ipUint[U]{^u.hi, ^u.lo}
}
