package netipds

import (
	"fmt"
	"net/netip"
)

// key stores the string of bits which represent the full path to a node in a
// prefix tree. The key is stored, big-endian, in the content field.
//
// offset stores the starting position of the key segment owned by the node.
//
// len measures the full length of the key starting from bit 0.
//
// The content field should not have any bits set beyond len (newKey enforces
// this).
type key[B keybits[B]] struct {
	len     uint8
	offset  uint8
	content B
}

// newKey returns a new key with the content truncated to length bits.
func newKey[B keybits[B]](content B, offset, length uint8) key[B] {
	return key[B]{length, offset, content.BitsClearedFrom(length)}
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
	return minU8(minU8(o.len, k.len), k.content.CommonPrefixLen(o.content))
}

// Rest returns a copy of k starting at position i.
//
// If i > k.len, then Rest returns the zero key.
func (k key[B]) Rest(i uint8) key[B] {
	if k.IsZero() || i > k.len {
		return key[B]{}
	}
	return newKey(k.content, i, k.len)
}

// IsZero reports whether k.len == 0.
func (k key[B]) IsZero() bool {
	return k.len == 0
}

// Truncated returns a copy of k with all content beyond the nth bit cleared.
func (k key[B]) Truncated(n uint8) key[B] {
	return newKey(k.content.BitsClearedFrom(n), k.offset, n)
}

// IsPrefixOf reports whether k is a prefix of o or is equal to o, i.e. k.len
// <= o.len and has the same content as o up to position k.len.
func (k key[B]) IsPrefixOf(o key[B]) bool {
	if k.len > o.len {
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
	return newKey(content, k.len, k.len+1)
}

// Rooted returns a copy of k with offset set to 0
func (k key[B]) Rooted() key[B] {
	return newKey(k.content, 0, k.len)
}

// ToPrefix returns the netip.Prefix that represents k.
func (k key[B]) ToPrefix() netip.Prefix {
	if k.IsZero() {
		return netip.Prefix{}
	}
	return netip.PrefixFrom(k.content.ToAddr(), int(k.len))
}

// key4FromPrefix returns the key that represents the provided Prefix.
func key4FromPrefix(p netip.Prefix) key[keybits4] {
	a4 := p.Addr().As4()
	return newKey(keybits4{beUint32(a4[:])}, 0, uint8(p.Bits()))
}

// key6FromPrefix returns the key that represents the provided Prefix.
func key6FromPrefix(p netip.Prefix) key[keybits6] {
	return newKey(u128From16(p.Addr().As16()), 0, uint8(p.Bits()))
}
