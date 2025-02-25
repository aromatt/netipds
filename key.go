package netipds

import (
	"net/netip"
)

type key struct {
	content uint128
	offset  uint8
	len     uint8
}

func newKey(content uint128, offset uint8, len uint8) key {
	return key{content.bitsClearedFrom(len), offset, len}
}

// keyFromPrefix returns the key that represents the provided Prefix.
func keyFromPrefix(p netip.Prefix) key {
	addr := p.Addr()
	// TODO bits could be -1
	bits := uint8(p.Bits())
	if addr.Is4() {
		bits = bits + 96
	}
	return newKey(u128From16(addr.As16()), 0, bits)
}

// toPrefix returns the Prefix represented by k.
func (k key) toPrefix() netip.Prefix {
	var a16 [16]byte
	bePutUint64(a16[:8], k.content.hi)
	bePutUint64(a16[8:], k.content.lo)
	addr := netip.AddrFrom16(a16)
	bits := int(k.len)
	if addr.Is4In6() {
		bits -= 96
	}
	return netip.PrefixFrom(addr.Unmap(), bits)
}

func (s key) bit(i uint8) bit {
	return s.content.isBitSet(i)
}

// isZero reports whether k is the zero key.
func (k key) isZero() bool {
	// Bits beyond len are always ignored, so if k.len == zero, then this
	// key effectively contains no bits.
	return k.len == 0
}

// rest returns a copy of k with offset = i.
//
// Returns the zero key if i > k.len or k.isZero().
func (k key) rest(i uint8) key {
	if k.isZero() || i > k.len {
		return key{}
	}
	return newKey(k.content, i, k.len)
}

// halves splits k into two halfkeys.
//
// Note: if k.offset > 64, then the hi half will be invalid.
func (k key) halves() (halfkey, halfkey) {
	return halfkey{k.content.hi, k.offset, 64}, halfkey{k.content.lo, 64, k.len}
}

// TODO: a key may have offset < 64 and len >= 64
func (k key) half() halfkey {
	// TODO can we make the type system prevent this?
	if (k.offset < 64) != (k.len < 64) {
		panic("tried to get half() of a key that crosses partitions")
	}
	if k.len < 64 {
		return halfkey{k.content.hi, k.offset, k.len}
	}
	return halfkey{k.content.lo, k.offset, k.len}
}

// halfkey returns the half of k that resides in the same partition as s.
// If k ends in lo and s ends in hi, then... TODO
//func (k key) half(h halfkey) halfkey {
//	if h.len > 64 {
//		return halfkey{k.content.lo, 64, k.len}
//}
