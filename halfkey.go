package netipds

import (
	"fmt"
	"strings"
)

// halfkey stores a string of bits which represent part of a path to a node in
// a prefix tree.
//
// The halfkey's bits are stored in the content field, between offset and len
// (counting from most-significant toward least-significant).
//
// Each halfkey's offset and len must both be in the range [0, 63] or
// [64, 127].
//
// The content field should not have any bits set beyond len. newHalfkey
// enforces this.
type halfkey struct {
	content uint64
	offset  uint8
	len     uint8
}

func newHalfkey(content uint64, offset uint8, len uint8) halfkey {
	return halfkey{bitsClearedFrom(content, len), offset, len}
}

// rooted returns a copy of s with offset set to 0
func (s halfkey) rooted() halfkey {
	return halfkey{s.content, 0, s.len}
}

// String prints the halfkey's content in hex, followed by "," + s.len. The
// least significant bit in the output is the bit at position (s.len - 1).
// Leading zeros are omitted.
func (s halfkey) String() string {
	var content string
	// TODO remove
	//just := k.content.shiftRight(128 - k.len)
	just := s.content >> (64 - s.len)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d", content, s.len)
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
func (s halfkey) StringRel() string {
	var content string
	//just := k.content.shiftLeft(k.offset).shiftRight(128 - k.len + k.offset)
	just := (s.content << s.offset) >> (64 - s.len + s.offset)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d", content, s.len-s.offset)
}

// truncated returns a copy of halfkey truncated to n bits.
func (s halfkey) truncated(n uint8) halfkey {
	return newHalfkey(s.content, s.offset, n)
}

// rest returns a copy of s starting at position i. if i > s.len, returns the
// zero halfkey.
func (s halfkey) rest(i uint8) halfkey {
	if s.isZero() {
		return s
	}
	if i > s.len {
		i = 0
	}
	return newHalfkey(s.content, i, s.len)
}

// TODO remove?
func isBitSet(u uint64, bit uint8) uint8 {
	return uint8(u >> (64 - bit) & 1)
}

func (s halfkey) bit(i uint8) bit {
	return bit(s.content >> (64 - i) & 1)
}

// equalFromRoot reports whether s and o have the same content and len (offsets
// are ignored).
func (s halfkey) equalFromRoot(o halfkey) bool {
	return s.len == o.len && s.content == o.content
}

// equalFullFromRoot reports whether s and k have the same content and len
// (offsets are ignored).
func (s halfkey) keyEqualFromRoot(k key) bool {
	return s.len == k.len && s.content == k.content.hi
}

// equalHalf reports whether s is equal to its respective half of f.
// TODO remove if unused
func (s halfkey) equalHalf(k key) bool {
	if s.offset < 64 {
		return s.content == k.content.hi
	}
	return s.content == k.content.lo
}

// commonPrefixLen returns the length of the common prefix between s and o,
// truncated to the length of the shorter of the two.
func (s halfkey) commonPrefixLen(o halfkey) (n uint8) {
	return min(min(o.len, s.len), u64CommonPrefixLen(s.content, o.content))
}

// keyCommonPrefixLen returns the length of the common prefix between s and k,
// truncated to the length of the shorter of the two.
func (s halfkey) keyCommonPrefixLen(k key) (n uint8) {
	return min(min(k.len, s.len), u64CommonPrefixLen(s.content, k.content.hi))
}

// isPrefixOf reports whether s has the same content as o up to position s.len.
//
// If strict, returns false if s == o.
func (s halfkey) isPrefixOf(o halfkey, strict bool) bool {
	if s.len <= o.len && s.content == bitsClearedFrom(o.content, s.len) {
		return !(strict && s.equalFromRoot(o))
	}
	return false
}

// isZero reports whether s is the zero halfkey.
func (s halfkey) isZero() bool {
	// Bits beyond len are always ignored, so if s.len == zero, then this
	// halfkey effectively contains no bits.
	return s.len == 0
}

// next returns a one-bit halfkey just beyond s, set to 1 if b == bitR.
func (s halfkey) next(b bit) (ret halfkey) {
	switch b {
	case bitL:
		ret = halfkey{
			content: s.content,
			offset:  s.len,
			len:     s.len + 1,
		}
	case bitR:
		ret = halfkey{
			// TODO remove
			//content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
			content: s.content | (uint64(1) << (64 - s.len - 1)),
			offset:  s.len,
			len:     s.len + 1,
		}
	}
	return
}
