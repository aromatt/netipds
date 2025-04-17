package netipds

import (
	"fmt"
	"net/netip"
)

type key[B KeyBits[B]] struct {
	len     uint8
	offset  uint8
	content B
}

func NewKey[B KeyBits[B]](content B, offset, len uint8) key[B] {
	return key[B]{len, offset, content}
}

// Bit returns the bit at position i in k.content.
func (k key[B]) Bit(i uint8) bit {
	return k.content.Bit(i)
}

// String prints the portion of k.content from offset to len, as hex,
// followed by ",<len>-<offset>". The least significant bit in the output is
// the bit at position (h.len - 1). Leading zeros are omitted.
func (k key[B]) String() string {
	return fmt.Sprintf("%s,%d-%d", k.content.Justify(k.offset, k.len), k.offset, k.len)
}

// Equal reports whether k and o have the same content and len.
func (k key[B]) EqualFromRoot(o key[B]) bool {
	return k.len == o.len && k.content == o.content
}

// CommonPrefixLen returns the length of the common prefix between k and o,
// truncated to the minimum of k.len and o.len.
func (k key[B]) CommonPrefixLen(o key[B]) uint8 {
	return min(min(o.len, k.len), k.content.CommonPrefixLen(o.content))
}

// Rest returns a copy of k starting at position i.
//
// If i > k.len, then Rest returns the zero key.
func (k key[B]) Rest(i uint8) key[B] {
	if k.IsZero() || i > k.len {
		return key[B]{}
	}
	return NewKey(k.content, i, k.len)
}

// IsZero reports whether k.len == 0.
func (k key[B]) IsZero() bool {
	return k.len == 0
}

// Truncated returns a copy of k truncated to n bits.
func (k key[B]) Truncated(n uint8) key[B] {
	return NewKey(k.content.BitsClearedFrom(n), k.offset, n)
}

// IsPrefixOf reports whether k is a prefix of o or is equal to o, i.e. k.len
// <= o.len and has the same content as o up to position k.len.
func (k key[B]) IsPrefixOf(o key[B]) bool {
	if k.len > o.len {
		return false
	}
	return k.content == o.content.BitsClearedFrom(k.len)
}

// IsPrefixOfStrict reports whether k is a strict prefix of o, i.e.
// k.len < o.len and has the same content as o up to position k.len.
func (k key[B]) IsPrefixOfStrict(o key[B]) bool {
	if k.len >= o.len {
		return false
	}
	return k.content == o.content.BitsClearedFrom(k.len)
}

// Next returns a one-bit key just beyond k, set to 1 iff b == bitR.
func (k key[B]) Next(b bit) key[B] {
	content := k.content
	if b == bitR {
		content = content.WithBitSet(k.len)
	}
	return NewKey(content, k.len, k.len+1)
}

// Rooted returns a copy of k with offset set to 0
func (k key[B]) Rooted() key[B] {
	return NewKey(k.content, 0, k.len)
}

// key4FromPrefix returns the key that represents the provided Prefix.
func key4FromPrefix(p netip.Prefix) key[keyBits4] {
	a4 := p.Addr().As4()
	return NewKey(keyBits4{beUint32(a4[:])}, 0, uint8(p.Bits()))
}

// key6FromPrefix returns the key that represents the provided Prefix.
func key6FromPrefix(p netip.Prefix) key[keyBits6] {
	addr := p.Addr()
	// TODO len could be -1
	len := uint8(p.Bits())
	return NewKey(u128From16(addr.As16()), 0, len)
}
