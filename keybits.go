package netipds

import (
	"fmt"
	"net/netip"
)

// bit is used as a selector for a node's children.
//
// bitL refers to the left child, and bitR to the right.
type bit = bool

const (
	bitL = false
	bitR = true
)

// keybits is an interface over integer types of different widths for use as
// keys in a binary radix tree.
//
// Using right-sized uints for IPv4 and IPv6 reduces memory usage by
// 22%, at the cost of a small hit to lookup-performance (~single-digit
// percentage), as measured by https://github.com/gaissmai/iprbench.
//
// The maximum intended width is 128 bits (see uint128.go).
//
// T must be the same as the implementing struct type itself.
type keybits[T comparable] interface {
	comparable

	// IsZero returns true if all bits are unset.
	IsZero() bool

	// BitsClearedFrom returns a copy of this keybits with all bits cleared
	// from bit position i to the end (big-endian).
	BitsClearedFrom(i uint8) T

	// Bit returns the bit at position i (big-endian).
	Bit(i uint8) bit

	// CommonPrefixLen returns the length of the common prefix between this key
	// and o (big-endian), truncated to the minimum of this key's length and
	// o's length.
	CommonPrefixLen(o T) uint8

	// WithBitSet returns a copy of this keybits with the bit at position i set
	// (big-endian).
	WithBitSet(i uint8) T

	// Justify returns a copy of this keybits shifted left by offset and right
	// by length.
	Justify(offset uint8, length uint8) T

	// String returns a string representation of the keybits.
	String() string

	// Uint128 returns the keybits as a uint128.
	//
	// This is used to offset the performance impact of using generics by
	// converting all keys to uint128 before hot loops.
	Uint128() uint128

	// ToAddr returns the keybits as a netip.Addr.
	ToAddr() netip.Addr
}

// keybits4 is a 32-bit keybits (for IPv4 keys).
type keybits4 uint32

func (k keybits4) IsZero() bool {
	return k == 0
}

func (k keybits4) BitsClearedFrom(bit uint8) keybits4 {
	return k >> (32 - bit) << (32 - bit)
}

func (k keybits4) Bit(i uint8) bit {
	return k&(1<<(31-i)) != 0
}

func (k keybits4) CommonPrefixLen(o keybits4) uint8 {
	return u32CommonPrefixLen(uint32(k), uint32(o))
}

func (k keybits4) WithBitSet(i uint8) keybits4 {
	return k | (1 << (31 - i))
}

func (k keybits4) Justify(o, l uint8) keybits4 {
	return (k << o) >> (32 - l + o)
}

func (k keybits4) String() string {
	if k.IsZero() {
		return "0"
	}
	return fmt.Sprintf("%x", uint32(k))
}

func (k keybits4) Uint128() uint128 {
	return uint128{uint64(k) << 32, 0}
}

func (k keybits4) ToAddr() netip.Addr {
	var a4 [4]byte
	bePutUint32(a4[:], uint32(k))
	return netip.AddrFrom4(a4)
}

// keybits6 is a 64-bit keybits (for IPv6 keys).
type keybits6 = uint128

func (k keybits6) IsZero() bool {
	return k.isZero()
}

func (k keybits6) BitsClearedFrom(bit uint8) keybits6 {
	return k.bitsClearedFrom(bit)
}

func (k keybits6) Bit(i uint8) bit {
	if i < 64 {
		return k.hi&(1<<(63-i)) != 0
	}
	return k.lo&(1<<(127-i)) != 0

}

func minU8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func (k keybits6) CommonPrefixLen(o keybits6) uint8 {
	return minU8(128, k.commonPrefixLen(o))
}

func (k keybits6) WithBitSet(i uint8) keybits6 {
	return k.or(uint128{0, 1}.shiftLeft(127 - i))
}

func (k keybits6) Justify(o, l uint8) keybits6 {
	return k.shiftLeft(o).shiftRight(128 - l + o)
}

func (k keybits6) String() string {
	var content string
	if k.IsZero() {
		return "0"
	}
	if k.hi > 0 {
		content = fmt.Sprintf("%x", k.hi)
	}
	if k.lo > 0 {
		if k.hi > 0 {
			content = fmt.Sprintf("%s%0*x", content, 16, k.lo)
		} else {
			content = fmt.Sprintf("%s%x", content, k.lo)
		}
	}
	return content
}

func (k keybits6) Uint128() uint128 {
	return k
}

func (k keybits6) ToAddr() netip.Addr {
	var a16 [16]byte
	bePutUint64(a16[:8], k.hi)
	bePutUint64(a16[8:], k.lo)
	return netip.AddrFrom16(a16)
}
