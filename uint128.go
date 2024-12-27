// Copied with modifications into netipds from net/netip

// Copyright 2020 The Inet.Af AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netipds

import (
	"encoding/binary"
	"math/bits"
)

// uint128 represents a uint128 using two uint64s.
//
// When the methods below mention a bit number, bit 0 is the most
// significant bit (in hi) and bit 127 is the lowest (lo&1).
type uint128 struct {
	hi uint64
	lo uint64
}

func u128From16(a [16]byte) uint128 {
	return uint128{
		binary.BigEndian.Uint64(a[:8]),
		binary.BigEndian.Uint64(a[8:]),
	}
}

// isZero reports whether u == 0.
//
// It's faster than u == (uint128{}) because the compiler (as of Go
// 1.15/1.16b1) doesn't do this trick and instead inserts a branch in
// its eq alg's generated code.
func (u uint128) isZero() bool { return u.hi|u.lo == 0 }

// and returns the bitwise AND of u and m (u&m).
func (u uint128) and(m uint128) uint128 {
	return uint128{u.hi & m.hi, u.lo & m.lo}
}

// or returns the bitwise OR of u and m (u|m).
func (u uint128) or(m uint128) uint128 {
	return uint128{u.hi | m.hi, u.lo | m.lo}
}

// not returns the bitwise NOT of u.
func (u uint128) not() uint128 {
	return uint128{^u.hi, ^u.lo}
}

// subOne returns u - 1.
func (u uint128) subOne() uint128 {
	lo, borrow := bits.Sub64(u.lo, 1, 0)
	return uint128{u.hi - borrow, lo}
}

// addOne returns u + 1.
func (u uint128) addOne() uint128 {
	lo, carry := bits.Add64(u.lo, 1, 0)
	return uint128{u.hi + carry, lo}
}

func u64CommonPrefixLen(a, b uint64) uint8 {
	return uint8(bits.LeadingZeros64(a ^ b))
}

func (u uint128) commonPrefixLen(v uint128) (n uint8) {
	if n = u64CommonPrefixLen(u.hi, v.hi); n == 64 {
		n += u64CommonPrefixLen(u.lo, v.lo)
	}
	return
}

// commonPrefixLenTrunc compares the first limit bits of u and v, returning the
// length of their common prefix within that portion.
func (u uint128) commonPrefixLenTrunc(v uint128, limit uint8) (n uint8) {
	if n = u64CommonPrefixLen(u.hi, v.hi); limit < n {
		return limit
	}
	if n == 64 {
		n += u64CommonPrefixLen(u.lo, v.lo)
	}
	return
}

// func (u *uint128) halves() [2]*uint64 {
// 	return [2]*uint64{&u.hi, &u.lo}
// }

// bitsSetFrom returns a copy of u with the given bit
// and all subsequent ones set.
func (u uint128) bitsSetFrom(bit uint8) uint128 {
	return u.or(mask6[bit].not())
}

// bitsClearedFrom returns a copy of u with the given bit
// and all subsequent ones cleared.
func (u uint128) bitsClearedFrom(bit uint8) uint128 {
	return u.and(mask6[bit])
}

// shiftRight returns a copy of u shifted right by the given
// number of bits.
func (u uint128) shiftRight(n uint8) uint128 {
	switch {
	case n == 0:
		return u
	case n < 64:
		return uint128{u.hi >> n, u.lo>>n | u.hi<<(64-n)}
	case n < 128:
		return uint128{0, u.hi >> (n - 64)}
	default:
		return uint128{}
	}
}

// shiftLeft returns a copy of u shifted left by the given
// number of bits.
func (u uint128) shiftLeft(n uint8) uint128 {
	switch {
	case n == 0:
		return u
	case n < 64:
		return uint128{u.hi<<n | u.lo>>(64-n), u.lo << n}
	case n < 128:
		return uint128{u.lo << (n - 64), 0}
	default:
		return uint128{}
	}
}

// isBitSet returns true if the bit at the given position is set.
// If bit > 127, returns false.
func (u uint128) isBitSet(bit uint8) bool {
	if bit < 64 {
		return u.hi&(uint64(1)<<(63-bit)) > 0
	} else {
		return u.lo&(uint64(1)<<(127-bit)) > 0
	}
}
