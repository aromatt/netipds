package netipds

import (
	"fmt"
	"net/netip"
)

//type key struct {
//	hi halfkey
//	lo halfkey
//}

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

// equalFromRoot reports whether k and o have the same content and len (offsets
// are ignored).
// TODO remove if not used
func (k key) equalFromRoot(o key) bool {
	return k.len == o.len && k.content == o.content
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

// halves splits k into two halfkeys: hi and lo.
//
// If k.offset > 64, then hi will be the zero halfkey.
// If k.len <= 64, then lo will be the zero halfkey.
func (k key) halves() (hi halfkey, lo halfkey) {
	if k.offset < 64 {
		hi = halfkey{k.content.hi, k.offset, 64}
	}
	if k.len > 64 {
		lo = halfkey{k.content.lo, 64, k.len}
	}
	return
}

// half returns the half of k in which k ends.
func (k key) endHalf() halfkey {
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

// next returns a one-bit key just beyond k, set to 1 if b == bitR.
// TODO remove if not used
func (k key) next(b bit) (ret key) {
	switch b {
	case bitL:
		ret = key{
			content: k.content,
			offset:  k.len,
			len:     k.len + 1,
		}
	case bitR:
		ret = key{
			content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
			offset:  k.len,
			len:     k.len + 1,
		}
	}
	return
}

// isPrefixOf reports whether k has the same content as o up to position k.len.
//
// If strict, returns false if k == o.
// TODO remove if not used
func (k key) isPrefixOf(o key, strict bool) bool {
	if k.len <= o.len && k.content == o.content.bitsClearedFrom(k.len) {
		return !(strict && k.equalFromRoot(o))
	}
	return false
}

// String prints the key's content in hex, followed by "," + s.len. The
// least significant bit in the output is the bit at position (s.len - 1).
// Leading zeros are omitted.
func (k key) String() string {
	var content string
	just := k.content.shiftRight(128 - k.len)
	if just.isZero() {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d", content, k.len)
}
