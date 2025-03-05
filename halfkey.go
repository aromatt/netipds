package netipds

import (
	"fmt"
	"strings"
)

// halfkey stores up to 64 bits, which represent up to half of a 128-bit key in
// a partitioned prefix tree.
//
// The halfkey's bits are stored in the content field. The content field is
// always aligned at bit 0 (the "hi" partition) or bit 64 (the "lo" partition).
// The offset and len fields imply in which partition the halfkey resides.
//
// The offset and len fields also specify which range of bits within the full
// 128-bit key are owned by the halfkey.
//
// The offset and len must both be in the range [0, 63] or [64, 127].
//
// The content field should not have any bits set beyond len (or len - 64, if
// len > 64). newHalfkey enforces this.
type halfkey struct {
	content uint64
	offset  uint8
	len     uint8
}

func bitsClearedFrom64(u uint64, bit uint8) uint64 {
	return u & mask64[bit]
}

func newHalfkey(content uint64, offset uint8, len uint8) halfkey {
	len64 := len
	if len > 64 {
		len64 -= 64
	}
	return halfkey{bitsClearedFrom64(content, len64), offset, len}
}

// rooted returns a copy of h with offset set to 0
func (h halfkey) rooted() halfkey {
	return halfkey{h.content, 0, h.len}
}

func (h halfkey) len64() uint8 {
	if h.len > 64 {
		return h.len - 64
	}
	return h.len
}

func (h halfkey) offset64() uint8 {
	if h.offset >= 64 {
		return h.offset - 64
	}
	return h.offset
}

// String prints the halfkey's content in hex, followed by ",<offset>-<len>".
// The least significant bit in the output is the bit at position (h.len - 1).
// Leading zeros are omitted.
func (h halfkey) String() string {
	var content string
	just := h.content >> (64 - h.len64())
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d-%d", content, h.offset, h.len)
}

// Parse parses the output of String.
// Parse is intended to be used only in tests.
func (h *halfkey) Parse(str string) error {
	var err error

	// Isolate content and len
	parts := strings.Split(str, ",")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse halfkey '%s': invalid format", h)
	}
	contentStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &h.len); err != nil {
		return fmt.Errorf("failed to parse halfkey '%s': %w", h, err)
	}

	lo := uint64(0)
	loStart := 0
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse halfkey: '%s', %w", h, err)
	}
	h.content = lo << (64 - h.len)
	h.offset = 0
	return nil
}

// StringRel prints the portion of h.content from h.offset to h.len, as hex,
// followed by ",<len>-<offset>". Leading zeros are omitted.
//
// This representation is lossy in that it hides the first h.offset bits, but
// it's helpful for debugging in the context of a pretty-printed tree.
func (h halfkey) StringRel() string {
	var content string
	just := (h.content << h.offset64()) >> (64 - h.len64() + h.offset64())
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d-%d", content, h.offset, h.len)
}

// truncated returns a copy of halfkey truncated to n bits.
func (h halfkey) truncated(n uint8) halfkey {
	return newHalfkey(h.content, h.offset, n)
}

// rest returns a copy of h starting at position i.
//
// Returns the zero halfkey if i > h.len or h.isZero().
func (h halfkey) rest(i uint8) halfkey {
	if h.isZero() || i > h.len {
		return halfkey{}
	}
	return newHalfkey(h.content, i, h.len)
}

// TODO remove?
func isBitSet(u uint64, bit uint8) uint8 {
	return uint8(u >> (63 - bit) & 1)
}

func (h halfkey) bit(i uint8) bit {
	if i >= 64 {
		i -= 64
	}
	return bit(h.content >> (63 - i) & 1)
}

// equalFromRoot reports whether h and o have the same content and len (offsets
// are ignored).
func (h halfkey) equalFromRoot(o halfkey) bool {
	return h.len == o.len && h.content == o.content
}

// keyEqualFromRoot reports whether h and k have the same content and len
// (offsets are ignored).
//
// Note: only keys with len <= 64 can be equal to a halfkey.
func (h halfkey) keyEqualFromRoot(k key) bool {
	return h.len == k.len && h.content == k.content.hi
}

// keyEndEqualFromRoot returns true if h and k have equal len, end in the same
// partition, equal content in the corresponding half of k.
// TODO does this obviate keyEqualFromRoot?
func (h halfkey) keyEndEqualFromRoot(k key) bool {
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
func (h halfkey) commonPrefixLen(o halfkey) (n uint8) {
	return min(min(o.len, h.len), u64CommonPrefixLen(h.content, o.content))
}

// keyHalfCommonPrefixLen compares h with the half of k in which h resides, and
// returns the length of the common prefix between them, truncated to the
// length of the shorter of the two.
//
// Panics if h is in the lo partition but k is only the hi partition, because
// we don't expect any caller to use this method in that scenario. TODO
func (h halfkey) keyHalfCommonPrefixLen(k key) (n uint8) {
	hiPad := uint8(0)
	kHalf := k.content.hi
	if h.len > 64 {
		if k.len <= 64 {
			panic("trying to compare prefix of lo halfkey with hi full key")
		}
		kHalf = k.content.lo
		hiPad = 64
	}
	return min(min(k.len, h.len), u64CommonPrefixLen(h.content, kHalf)+hiPad)
}

// isPrefixOf reports whether h has the same content as o up to position h.len.
//
// If strict, returns false if h == o.
func (h halfkey) isPrefixOf(o halfkey, strict bool) bool {
	if h.len <= o.len && h.content == bitsClearedFrom64(o.content, h.len64()) {
		return !(strict && h.equalFromRoot(o))
	}
	return false
}

// isPrefixOfKeyEnd reports whether h has the same content as its counterpart
// half of k up to position h.len.
//
// If strict, returns false if h == <k half>.
func (h halfkey) isPrefixOfKeyEnd(k key, strict bool) bool {
	if h.len > k.len {
		return false
	}
	kHalf := k.content.hi
	if h.len > 64 {
		kHalf = k.content.lo
	}
	if h.content == bitsClearedFrom64(kHalf, h.len64()) {
		return !(strict && h.content == kHalf && h.len == k.len)
	}
	return false
}

// isZero reports whether h is the zero halfkey.
func (h halfkey) isZero() bool {
	// Bits beyond len are always ignored, so if h.len == zero, then this
	// halfkey effectively contains no bits.
	return h.len == 0
}

// next returns a one-bit halfkey just beyond s, set to 1 if b == bitR.
// TODO should/does this handle partition-crossing?
func (h halfkey) next(b bit) (ret halfkey) {
	switch b {
	case bitL:
		ret = halfkey{
			content: h.content,
			offset:  h.len,
			len:     h.len + 1,
		}
	case bitR:
		ret = halfkey{
			// TODO remove
			//content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
			content: h.content | (uint64(1) << (64 - h.len - 1)),
			offset:  h.len,
			len:     h.len + 1,
		}
	}
	return
}
