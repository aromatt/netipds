package netipds

import (
	"fmt"
	"net/netip"
	"strings"
)

// key4 stores the string of bits which represent the full path to a node in a
// prefix tree. The maximum length is 64 bits. These bits are stored in the
// most-significant bits of the content field.
//
// offset stores the starting position of the key segment owned by the node.
//
// len measures the full length of the key starting from bit 0.
//
// The content field should not have any bits set beyond len. newKey enforces
// this.
type key4 struct {
	content uint64
	offset  uint8
	len     uint8
}

func newKey4(content uint64, offset uint8, len uint8) key4 {
	len64 := len
	if len64 > 64 {
		len64 -= 64
	}
	return key4{bitsClearedFrom(content, len64), offset, len}
}

// rooted returns a copy of h with offset set to 0
func (h key4) rooted() key4 {
	return key4{h.content, 0, h.len}
}

func u64From4(a [4]byte) uint64 {
	return uint64(a[0])<<24 | uint64(a[1])<<16 | uint64(a[2])<<8 | uint64(a[3])
}

// key4FromPrefix returns the key that represents the provided Prefix.
func key4FromPrefix(p netip.Prefix) key4 {
	addr := p.Addr()
	// TODO bits could be -1
	bits := uint8(p.Bits())
	if addr.Is4() {
		bits = bits + 96
	}
	return newKey4(u64From4(addr.As4()), 0, bits)
}

// String prints the key4's content in hex, followed by ",<offset>-<len>".
// The least significant bit in the output is the bit at position (h.len - 1).
// Leading zeros are omitted.
func (h key4) String() string {
	var content string
	// TODO remove
	//just := k.content.shiftRight(128 - k.len)
	just := h.content >> (64 - h.len)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d-%d", content, h.offset, h.len)
}

// Parse parses the output of String.
// Parse is intended to be used only in tests.
func (h *key4) Parse(str string) error {
	var err error

	// Isolate content and len
	parts := strings.Split(str, ",")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse key4 '%s': invalid format", h)
	}
	contentStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &h.len); err != nil {
		return fmt.Errorf("failed to parse key4 '%s': %w", h, err)
	}

	lo := uint64(0)
	loStart := 0
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse key4: '%s', %w", h, err)
	}
	h.content = lo << (64 - h.len)
	h.offset = 0
	return nil
}

// StringRel prints the portion of h.content from offset to len, as hex,
// followed by ",<len>-<offset>". The least significant bit in the output is
// the bit at position (h.len - 1). Leading zeros are omitted.
//
// This representation is lossy in that it hides the first h.offset bits, but
// it's helpful for debugging in the context of a pretty-printed tree.
func (h key4) StringRel() string {
	var content string
	just := (h.content << h.offset) >> (64 - h.len + h.offset)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d-%d", content, h.offset, h.len)
}

// truncated returns a copy of key4 truncated to n bits.
func (h key4) truncated(n uint8) key4 {
	return newKey4(h.content, h.offset, n)
}

// rest returns a copy of h starting at position i.
//
// Returns the zero key4 if i > h.len or h.isZero().
func (h key4) rest(i uint8) key4 {
	if h.isZero() || i > h.len {
		return key4{}
	}
	return newKey4(h.content, i, h.len)
}

// TODO remove?
func isBitSet(u uint64, bit uint8) uint8 {
	return uint8(u >> (64 - bit) & 1)
}

func (h key4) bit(i uint8) bit {
	return bit(h.content >> (64 - i) & 1)
}

// equalFromRoot reports whether h and o have the same content and len (offsets
// are ignored).
func (h key4) equalFromRoot(o key4) bool {
	return h.len == o.len && h.content == o.content
}

// keyEqualFromRoot reports whether h and k have the same content and len
// (offsets are ignored).
//
// Note: only keys with len <= 64 can be equal to a key4.
func (h key4) keyEqualFromRoot(k key) bool {
	return h.len == k.len && h.content == k.content.hi
}

// keyEndEqualFromRoot returns true if h and k have equal len, end in the same
// partition, equal content in the corresponding half of k.
// TODO does this obviate keyEqualFromRoot?
func (h key4) keyEndEqualFromRoot(k key) bool {
	if h.len != k.len {
		return false
	}
	if h.len <= 64 {
		return h.content == k.content.hi
	}
	return h.content == k.content.lo
}

// commonPrefixLen returns the length of the common prefix between h and o,
// truncated to the length of the shorter of the two.
func (h key4) commonPrefixLen(o key4) (n uint8) {
	return min(min(o.len, h.len), u64CommonPrefixLen(h.content, o.content))
}

// keyHalfCommonPrefixLen compares h with the half of k in which h resides, and
// returns the length of the common prefix between them, truncated to the
// length of the shorter of the two.
//
// Panics if h is in the lo partition but k is only the hi partition, because
// we don't expect any caller to use this method in that scenario. TODO
//func (h key4) keyHalfCommonPrefixLen(k key) (n uint8) {
//	hiPad := uint8(0)
//	kHalf := k.content.hi
//	if h.len > 64 {
//		if k.len <= 64 {
//			panic("trying to compare prefix of lo key4 with hi full key")
//		}
//		kHalf = k.content.lo
//		hiPad = 64
//	}
//	return min(min(k.len, h.len), u64CommonPrefixLen(h.content, kHalf)+hiPad)
//}

// isPrefixOf reports whether h has the same content as o up to position h.len.
//
// If strict, returns false if h == o.
func (h key4) isPrefixOf(o key4, strict bool) bool {
	if h.len <= o.len && h.content == bitsClearedFrom(o.content, h.len) {
		return !(strict && h.equalFromRoot(o))
	}
	return false
}

// isPrefixOfKeyEnd reports whether h has the same content as its counterpart
// half of k up to position h.len.
//
// If strict, returns false if h == <k half>.
func (h key4) isPrefixOfKeyEnd(k key, strict bool) bool {
	if h.len > k.len {
		return false
	}
	kHalf := k.content.hi
	len64 := h.len
	if h.len > 64 {
		kHalf = k.content.lo
		len64 -= 64
	}
	if h.content == bitsClearedFrom(kHalf, len64) {
		return !(strict && h.content == kHalf && h.len == k.len)
	}
	return false
}

// isZero reports whether h is the zero key4.
func (h key4) isZero() bool {
	// Bits beyond len are always ignored, so if h.len == zero, then this
	// key4 effectively contains no bits.
	return h.len == 0
}

// next returns a one-bit key4 just beyond s, set to 1 if b == bitR.
// TODO should/does this handle partition-crossing?
func (h key4) next(b bit) (ret key4) {
	switch b {
	case bitL:
		ret = key4{
			content: h.content,
			offset:  h.len,
			len:     h.len + 1,
		}
	case bitR:
		ret = key4{
			// TODO remove
			//content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
			content: h.content | (uint64(1) << (64 - h.len - 1)),
			offset:  h.len,
			len:     h.len + 1,
		}
	}
	return
}
