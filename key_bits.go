package netipds

import (
	"fmt"
)

type KeyBits[T comparable] interface {
	comparable
	IsZero() bool
	BitsClearedFrom(uint8) T
	Bit(uint8) bit
	CommonPrefixLen(T) uint8
	// TODO For use by Next()
	WithBitSet(uint8) T
	// TODO For use by StringRel()
	Justify(uint8, uint8) T
	String() string
	To128() uint128
}

type keyBits4 struct {
	bits uint32
}

func (k keyBits4) IsZero() bool {
	return k.bits == 0
}

func (k keyBits4) BitsClearedFrom(bit uint8) keyBits4 {
	return keyBits4{k.bits >> (32 - bit) << (32 - bit)}
}

func (k keyBits4) Bit(i uint8) bit {
	return k.bits&(1<<(31-i)) != 0
}

func (k keyBits4) CommonPrefixLen(o keyBits4) uint8 {
	return u32CommonPrefixLen(k.bits, o.bits)
}

func (k keyBits4) WithBitSet(i uint8) keyBits4 {
	return keyBits4{k.bits | (1 << (31 - i))}
}

// TODO
func (k keyBits4) Justify(o, l uint8) keyBits4 {
	return keyBits4{(k.bits << o) >> (32 - l + o)}
}

func (k keyBits4) String() string {
	if k.IsZero() {
		return "0"
	}
	return fmt.Sprintf("%x", k.bits)
}

func (k keyBits4) To128() uint128 {
	return uint128{uint64(k.bits) << 32, 0}
}

type keyBits6 = uint128

func (k keyBits6) IsZero() bool {
	return k.isZero()
}

func (k keyBits6) BitsClearedFrom(bit uint8) keyBits6 {
	return k.bitsClearedFrom(bit)
}

func (k keyBits6) Bit(i uint8) bit {
	return k.isBitSet(i)
}

func (k keyBits6) CommonPrefixLen(o keyBits6) uint8 {
	return min(min(128, 128), k.commonPrefixLen(o))
}

func (k keyBits6) WithBitSet(i uint8) keyBits6 {
	return k.or(uint128{0, 1}.shiftLeft(127 - i))
}

// TODO
func (k keyBits6) Justify(o, l uint8) keyBits6 {
	return k.shiftLeft(o).shiftRight(128 - l + o)
}

func (k keyBits6) String() string {
	var content string
	if k.IsZero() {
		return "0"
	}
	if k.hi > 0 {
		content = fmt.Sprintf("%x", k.hi)
	}
	if k.lo > 0 {
		content = fmt.Sprintf("%s%x", content, k.lo)
	}
	return fmt.Sprintf("%s", content)
}

func (k keyBits6) To128() uint128 {
	return k
}
