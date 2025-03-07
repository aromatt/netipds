package netipds

import (
	"fmt"
	"math/bits"
	"net/netip"
)

type BitOps[U KeyBits] interface {
	// IsZero reports whether u == 0.
	IsZero(U) bool
	// BitsClearedFrom returns a copy of u with all bits starting from b set to 0.
	BitsClearedFrom(u U, b uint8) U
	// Bit returns bitR if the bit at i is set, else bitL
	Bit(u U, i uint8) bit
	// BitBool returns true if the bit at i is set.
	BitBool(u U, i uint8) bool
	// CommonPrefixLen returns the number of leading bits that are the same in a and b.
	CommonPrefixLen(a U, b U) uint8
	// WithBitSet returns a copy of u with the given bit set.
	WithBitSet(u U, i uint8) U
	// String returns a string representation of u.
	String(U) string
	// Justify returns a copy of u containing only the bits between i and j,
	// justified to the right.
	Justify(u U, i, j uint8) U
	KeyBitsFromPrefix(prefix netip.Prefix) U
}

func NewBitOps[U KeyBits]() BitOps[U] {
	switch any(*new(U)).(type) {
	case uint32:
		return bitOps4{}
	case uint128:
		return bitOps6{}
	default:
		panic("unsupported type")
	}
}

type bitOps4 struct{}

func (bitOps4) IsZero(u uint32) bool {
	return u == 0
}

func (bitOps4) BitsClearedFrom(u uint32, bit uint8) uint32 {
	return u >> (32 - bit) << (32 - bit)
}

func (bitOps4) Bit(u uint32, i uint8) bit {
	return bit(u >> (31 - i) & 1)
}

func (bitOps4) BitBool(u uint32, i uint8) bool {
	return u&(1<<(31-i)) != 0
}

func (bitOps4) CommonPrefixLen(a, b uint32) uint8 {
	return uint8(bits.LeadingZeros32(a ^ b))
}

func (bitOps4) WithBitSet(u uint32, i uint8) uint32 {
	return u | (1 << (31 - i))
}

func (bitOps4) Justify(u uint32, i, j uint8) uint32 {
	return (u << i) >> (32 - j + i)
}

func (bitOps4) String(u uint32) string {
	if u == 0 {
		return "0"
	}
	return fmt.Sprintf("%x", u)
}

func (bitOps4) KeyBitsFromPrefix(p netip.Prefix) uint32 {
	a4 := p.Addr().As4()
	return beUint32(a4[:])
}

type bitOps6 struct{}

func (bitOps6) IsZero(u uint128) bool {
	return u.isZero()
}

func (bitOps6) BitsClearedFrom(u uint128, bit uint8) uint128 {
	return u.bitsClearedFrom(bit)
}

func (bitOps6) Bit(u uint128, i uint8) bit {
	return bit(u.isBitSet(i))
}

func (bitOps6) BitBool(u uint128, i uint8) bool {
	if i < 64 {
		return u.hi&(1<<(63-i)) != 0
	}
	return u.lo&(1<<(127-i)) != 0
}

func (bitOps6) CommonPrefixLen(a, b uint128) uint8 {
	return a.commonPrefixLen(b)
}

func (bitOps6) WithBitSet(u uint128, i uint8) uint128 {
	return u.or(uint128{0, 1}.shiftLeft(127 - i))
}

func (bitOps6) Justify(u uint128, i, j uint8) uint128 {
	return u.shiftLeft(i).shiftRight(128 - j + i)
}

func (bitOps6) String(u uint128) string {
	var content string
	if u.isZero() {
		return "0"
	}
	if u.hi > 0 {
		content = fmt.Sprintf("%x", u.hi)
	}
	if u.lo > 0 {
		if u.hi > 0 {
		} else {
			content = fmt.Sprintf("%s%x", content, u.lo)
		}
	}
	return fmt.Sprintf("%s", content)
}

func (bitOps6) KeyBitsFromPrefix(p netip.Prefix) uint128 {
	addr := p.Addr()
	// TODO len could be -1
	len := uint8(p.Bits())
	// TODO we shouldn't need to do this anymore
	if addr.Is4() {
		len = len + 96
	}
	return u128From16(addr.As16())
}
