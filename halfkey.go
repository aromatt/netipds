package netipds

import (
	"fmt"
	"strings"
)

// halfkey stores up to 64 bits, which represent up to half of a 128-bit key in
// a partitioned prefix tree. The content field is left-aligned at either bit 0
// or 64.
//
// The halfkey's bits are stored in the content field, which is left-aligned at
// either bit 0 or 64, between offset and len (counting from most-significant
// toward least-significant).
//
// The offset and len fields specify which range of bits within the full
// 128-bit key are owned by the halfkey. The offset and len must both be in the
// range [0, 63] or [64, 127].
//
// The content field should not have any bits set beyond len (or len - 64, if
// len > 64). newHalfkey enforces this.
type halfkey struct {
	content uint64
	offset  uint8
	len     uint8
}

func newHalfkey(content uint64, offset uint8, len uint8) halfkey {
	len64 := len
	if len64 > 64 {
		len64 -= 64
	}
	return halfkey{bitsClearedFrom(content, len64), offset, len}
}

// rooted returns a copy of h with offset set to 0
func (h halfkey) rooted() halfkey {
	return halfkey{h.content, 0, h.len}
}

// String prints the halfkey's content in hex, followed by "," + s.len. The
// least significant bit in the output is the bit at position (s.len - 1).
// Leading zeros are omitted.
func (h halfkey) String() string {
	var content string
	// TODO remove
	//just := k.content.shiftRight(128 - k.len)
	just := h.content >> (64 - h.len)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d", content, h.len)
}

// Parse parses the output of String.
// Parse is intended to be used only in tests.
func (s *halfkey) Parse(str string) error {
	var err error

	// Isolate content and len
	parts := strings.Split(str, ",")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse halfkey '%s': invalid format", s)
	}
	contentStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &s.len); err != nil {
		return fmt.Errorf("failed to parse halfkey '%s': %w", s, err)
	}

	lo := uint64(0)
	loStart := 0
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse halfkey: '%s', %w", s, err)
	}
	s.content = lo << (64 - s.len)
	s.offset = 0
	return nil
}

// StringRel prints the portion of s.content from offset to len, as hex,
// followed by "," + (len-offset). The least significant bit in the output is
// the bit at position (s.len - 1). Leading zeros are omitted.
//
// This representation is lossy in that it hides the first s.offset bits, but
// it's helpful for debugging in the context of a pretty-printed tree.
//
// TODO
//   - key{uint128{0, 1}, 127, 128} => "1,128"
//   - key{uint128{0, 2}, 126, 128} => "2,128"
//   - key{uint128{0, 2}, 126, 127} => "1,127"
//   - key{uint128{1, 1}, 63, 128} => "10000000000000001,128"
//   - key{uint128{1, 0}, 63, 64}  => "1,64"
//   - key{uint128{256, 0}, 56} => "1,56"
//   - key{uint128{256, 0}, 64} => "100,64"
func (h halfkey) StringRel() string {
	var content string
	//just := k.content.shiftLeft(k.offset).shiftRight(128 - k.len + k.offset)
	just := (h.content << h.offset) >> (64 - h.len + h.offset)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d", content, h.len-h.offset)
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
	return uint8(u >> (64 - bit) & 1)
}

func (h halfkey) bit(i uint8) bit {
	return bit(h.content >> (64 - i) & 1)
}

// equalFromRoot reports whether h and o have the same content and len (offsets
// are ignored).
func (h halfkey) equalFromRoot(o halfkey) bool {
	return h.len == o.len && h.content == o.content
}

// equalFullFromRoot reports whether h and k have the same content and len
// (offsets are ignored).
func (h halfkey) keyEqualFromRoot(k key) bool {
	return h.len == k.len && h.content == k.content.hi
}

// equalHalf reports whether h is equal to its respective half of f.
// TODO remove if unused
func (h halfkey) equalHalf(k key) bool {
	if h.offset < 64 {
		return h.content == k.content.hi
	}
	return h.content == k.content.lo
}

// commonPrefixLen returns the length of the common prefix between h and o,
// truncated to the length of the shorter of the two.
func (h halfkey) commonPrefixLen(o halfkey) (n uint8) {
	return min(min(o.len, h.len), u64CommonPrefixLen(h.content, o.content))
}

// keyCommonPrefixLen returns the length of the common prefix between h and k,
// truncated to the length of the shorter of the two.
func (h halfkey) keyCommonPrefixLen(k key) (n uint8) {
	return min(min(k.len, h.len), u64CommonPrefixLen(h.content, k.content.hi))
}

// isPrefixOf reports whether h has the same content as o up to position s.len.
//
// If strict, returns false if h == o.
func (h halfkey) isPrefixOf(o halfkey, strict bool) bool {
	if h.len <= o.len && h.content == bitsClearedFrom(o.content, h.len) {
		return !(strict && h.equalFromRoot(o))
	}
	return false
}

// isZero reports whether h is the zero halfkey.
func (h halfkey) isZero() bool {
	// Bits beyond len are always ignored, so if s.len == zero, then this
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
