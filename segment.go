package netipds

import (
	"fmt"
	"strings"
)

// segment stores a string of bits which represent part of a path to a node in
// a prefix tree.
//
// The segment's bits are stored in the content field, between offset and len
// (counting from most-significant toward least-significant).
//
// Each segment's offset and len must both be in the range [0, 63] or
// [64, 127].
//
// The content field should not have any bits set beyond len. newSegment
// enforces this.
type segment struct {
	content uint64
	offset  uint8
	len     uint8
}

func newSegment(content uint64, offset uint8, len uint8) segment {
	return segment{bitsClearedFrom(content, len), offset, len}
}

// rooted returns a copy of s with offset set to 0
func (s segment) rooted() segment {
	return segment{s.content, 0, s.len}
}

// String prints the segments's content in hex, followed by "," + s.len. The
// least significant bit in the output is the bit at position (s.len - 1).
// Leading zeros are omitted.
func (s segment) String() string {
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
func (s *segment) Parse(str string) error {
	var err error

	// Isolate content and len
	parts := strings.Split(str, ",")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse segment '%s': invalid format", s)
	}
	contentStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &s.len); err != nil {
		return fmt.Errorf("failed to parse segment '%s': %w", s, err)
	}

	lo := uint64(0)
	loStart := 0
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse segment: '%s', %w", s, err)
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
func (s segment) StringRel() string {
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

// truncated returns a copy of segment truncated to n bits.
func (s segment) truncated(n uint8) segment {
	return newSegment(s.content, s.offset, n)
}

// rest returns a copy of s starting at position i. if i > s.len, returns the
// zero segment.
func (s segment) rest(i uint8) segment {
	if s.isZero() {
		return s
	}
	if i > s.len {
		i = 0
	}
	return newSegment(s.content, i, s.len)
}

// TODO remove?
func isBitSet(u uint64, bit uint8) uint8 {
	return uint8(u >> (64 - bit) & 1)
}

func (s segment) bit(i uint8) bit {
	return bit(s.content >> (64 - i) & 1)
}

// equalFromRoot reports whether s and o have the same content and len (offsets
// are ignored).
func (s segment) equalFromRoot(o segment) bool {
	return s.len == o.len && s.content == o.content
}

// equalFullFromRoot reports whether s and k have the same content and len
// (offsets are ignored).
func (s segment) keyEqualFromRoot(k key) bool {
	return s.len == k.len && s.content == k.content.hi
}

// equalHalf reports whether s is equal to its respective half of f.
// TODO remove if unused
func (s segment) equalHalf(k key) bool {
	if s.offset < 64 {
		return s.content == k.content.hi
	}
	return s.content == k.content.lo
}

// commonPrefixLen returns the length of the common prefix between s and o,
// truncated to the length of the shorter of the two.
func (s segment) commonPrefixLen(o segment) (n uint8) {
	return min(min(o.len, s.len), u64CommonPrefixLen(s.content, o.content))
}

// keyCommonPrefixLen returns the length of the common prefix between s and k,
// truncated to the length of the shorter of the two.
func (s segment) keyCommonPrefixLen(k key) (n uint8) {
	return min(min(k.len, s.len), u64CommonPrefixLen(s.content, k.content.hi))
}

// isPrefixOf reports whether s has the same content as o up to position s.len.
//
// If strict, returns false if s == o.
func (s segment) isPrefixOf(o segment, strict bool) bool {
	if s.len <= o.len && s.content == bitsClearedFrom(o.content, s.len) {
		return !(strict && s.equalFromRoot(o))
	}
	return false
}

// isZero reports whether s is the zero segment.
func (s segment) isZero() bool {
	// Bits beyond len are always ignored, so if s.len == zero, then this
	// segment effectively contains no bits.
	return s.len == 0
}

// next returns a one-bit segment just beyond s, set to 1 if b == bitR.
func (s segment) next(b bit) (ret segment) {
	switch b {
	case bitL:
		ret = segment{
			content: s.content,
			offset:  s.len,
			len:     s.len + 1,
		}
	case bitR:
		ret = segment{
			// TODO remove
			//content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
			content: s.content | (uint64(1) << (64 - s.len - 1)),
			offset:  s.len,
			len:     s.len + 1,
		}
	}
	return
}
