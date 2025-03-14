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

func NewKey[B KeyBits[B]](b B, offset, len uint8) key[B] {
	return key[B]{len, offset, b}
}

func (k key[B]) Bit(i uint8) bit {
	return k.content.Bit(i)
}

// StringRel prints the portion of h.content from offset to len, as hex,
// followed by ",<len>-<offset>". The least significant bit in the output is
// the bit at position (h.len - 1). Leading zeros are omitted.
//
// This representation is lossy in that it hides the first h.offset bits, but
// it's helpful for debugging in the context of a pretty-printed tree.
func (k key[B]) StringRel() string {
	return fmt.Sprintf("%s,%d-%d",
		k.content.Justify(k.offset, k.len),
		k.offset, k.len)
}

func (k key[B]) EqualFromRoot(o key[B]) bool {
	return k.len == o.len && k.content == o.content
}

func (k key[B]) CommonPrefixLen(o key[B]) uint8 {
	return min(min(o.len, k.len), k.content.CommonPrefixLen(o.content))
}

func (k key[B]) Rest(i uint8) key[B] {
	if k.IsZero() || i > k.len {
		return key[B]{}
	}
	return NewKey(k.content, i, k.len)
}

func (k key[B]) IsZero() bool {
	return k.len == 0
}

func (k key[B]) Truncated(n uint8) key[B] {
	return NewKey(k.content.BitsClearedFrom(n), k.offset, n)
}

func (k key[B]) IsPrefixOf(o key[B]) bool {
	if k.len > o.len {
		return false
	}
	return k.content == o.content.BitsClearedFrom(k.len)
}

func (k key[B]) IsPrefixOfStrict(o key[B]) bool {
	if k.len >= o.len {
		return false
	}
	return k.content == o.content.BitsClearedFrom(k.len)
}

func (k key[B]) Next(b bit) key[B] {
	content := k.content
	if b == bitR {
		content = content.WithBitSet(k.len)
	}
	return NewKey(content, k.offset, k.len+1)
}

func (k key[B]) PathNext(path key[B]) bit {
	return path.Bit(k.len)
}

func (k key[B]) Rooted() key[B] {
	return NewKey(k.content, 0, k.len)
}

func (k key[B]) To128() key[keyBits6] {
	return key[uint128]{k.len, k.offset, k.content.To128()}
}

// key4FromPrefix4 returns the key that represents the provided Prefix.
func key4FromPrefix4(p netip.Prefix) key[keyBits4] {
	a4 := p.Addr().As4()
	return NewKey(keyBits4{beUint32(a4[:])}, 0, uint8(p.Bits()))
}

// TODO
func key6FromPrefix4(p netip.Prefix) key[keyBits6] {
	return NewKey(u128From16(p.Addr().As16()).shiftLeft(96), 0, uint8(p.Bits()))
}

// key6FromPrefix6 returns the key that represents the provided Prefix.
func key6FromPrefix6(p netip.Prefix) key[keyBits6] {
	return NewKey(u128From16(p.Addr().As16()), 0, uint8(p.Bits()))
}
