package netipds

import (
	"fmt"
	"net/netip"
)

type KeyOps[B KeyBits] struct {
	bo BitOps[B]
}

func NewKeyOps[B KeyBits]() KeyOps[B] {
	return KeyOps[B]{NewBitOps[B]()}
}

func (ko KeyOps[B]) NewKey(content B, offset, len uint8) key[B] {
	return key[B]{len, offset, ko.bo.BitsClearedFrom(content, len)}
}

func (ko KeyOps[B]) Rest(k key[B], i uint8) key[B] {
	if k.IsZero() || i > k.len {
		return key[B]{}
	}
	return ko.NewKey(k.content, i, k.len)

}

func (ko KeyOps[B]) Bit(k key[B], i uint8) bit {
	return ko.bo.Bit(k.content, i)
}

func (ko KeyOps[B]) BitBool(k key[B], i uint8) bool {
	return ko.bo.BitBool(k.content, i)
}

func (ko KeyOps[B]) Truncated(k key[B], n uint8) key[B] {
	return ko.NewKey(k.content, k.offset, n)
}

func (ko KeyOps[B]) Next(k key[B], b bit) key[B] {
	content := k.content
	if b == bitR {
		content = ko.bo.WithBitSet(content, k.len)
	}
	return ko.NewKey(content, k.offset, k.len+1)
}

func (ko KeyOps[B]) IsPrefixOf(k, o key[B]) bool {
	if k.len > o.len {
		return false
	}
	return k.content == ko.bo.BitsClearedFrom(o.content, k.len)
}

func (ko KeyOps[B]) IsPrefixOfStrict(k, o key[B]) bool {
	if k.len >= o.len {
		return false
	}
	return k.content == ko.bo.BitsClearedFrom(o.content, k.len)
}

func (ko KeyOps[B]) CommonPrefixLen(a, b key[B]) uint8 {
	return min(min(a.len, b.len), ko.bo.CommonPrefixLen(a.content, b.content))
}

func (ko KeyOps[B]) KeyFromPrefix(p netip.Prefix) key[B] {
	// TODO
	//	if addr.Is4() {
	//		len = len + 96
	//	}
	return ko.NewKey(ko.bo.KeyBitsFromPrefix(p), 0, uint8(p.Bits()))
}

func (ko KeyOps[B]) StringRel(k key[B]) string {
	return fmt.Sprintf("%x,%d-%d",
		ko.bo.Justify(k.content, k.offset, k.len),
		k.offset,
		k.len)
}
