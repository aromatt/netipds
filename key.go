package netipds

import (
	"fmt"
	"net/netip"
	"strings"
)

// key stores the bits which represent the full path to a node in a prefix
// tree. The maximum size of a key is 128 bits. The key is stored in the
// most-significant bits of the content field.
//
// offset defines the starting position of the key segment owned by the node.
//
// len measures the full length of the prefix from the root to the end of the
// node's segment.
//
// The content field should not have any bits set beyond len. newKey enforces
// this.
type key struct {
	content uint128
	offset  uint8
	len     uint8
}

func newKey(content uint128, offset uint8, len uint8) key {
	return key{content.bitsClearedFrom(len), offset, len}
}

// rooted returns a copy of key with offset set to 0
func (k key) rooted() key {
	return key{k.content, 0, k.len}
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

// String prints the key's content in hex, followed by "/" + k.len. The least
// significant bit in the output is the bit at position (k.len - 1). Leading
// zeros are omitted.
func (k key) String() string {
	var ret string
	just := k.content.shiftRight(128 - k.len)
	if just.isZero() {
		ret = "0"
	} else {
		if just.hi > 0 {
			ret = fmt.Sprintf("%x", just.hi)
		}
		if just.lo > 0 {
			if just.hi > 0 {
				ret = fmt.Sprintf("%s%0*x", ret, (k.len-64)/4, just.lo)
			} else {
				ret = fmt.Sprintf("%s%x", ret, just.lo)
			}
		}
	}
	return fmt.Sprintf("%s/%d", ret, k.len)
}

// Parse parses the output of String.
// Parse is intended to be used only in tests.
func (k *key) Parse(s string) error {
	var err error

	// Isolate content and len
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse key '%s': invalid format", s)
	}
	contentStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &k.len); err != nil {
		return fmt.Errorf("failed to parse key '%s': %w", s, err)
	}

	// lo = right-most 64 bits
	// hi = anything to the left of lo
	hi, lo := uint64(0), uint64(0)
	loStart := 0
	if len(contentStr) > 16 {
		loStart = len(contentStr) - 16
		if _, err = fmt.Sscanf(contentStr[:loStart], "%x", &hi); err != nil {
			return fmt.Errorf("failed to parse key: '%s', %w", s, err)
		}
	}
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse key: '%s', %w", s, err)
	}
	k.content = uint128{hi, lo}.shiftLeft(128 - k.len)
	k.offset = 0
	return nil
}

// StringRel prints the portion of k.content from offset to len, as hex,
// followed by "/" + (len-offset). The least significant bit in the output is
// the bit at position (k.len - 1). Leading zeros are omitted.
//
// This representation is lossy in that it hides the first k.offset bits, but
// it's helpful for debugging in the context of a pretty-printed tree.
//
//   - key{uint128{0, 1}, 127, 128} => "1/128"
//   - key{uint128{0, 2}, 126, 128} => "2/128"
//   - key{uint128{0, 2}, 126, 127} => "1/127"
//   - key{uint128{1, 1}, 63, 128} => "10000000000000001/128"
//   - key{uint128{1, 0}, 63, 64}  => "1/64"
//   - key{uint128{256, 0}, 56} => "1/56"
//   - key{uint128{256, 0}, 64} => "100/64"
func (k key) StringRel() string {
	var ret string
	just := k.content.shiftLeft(k.offset).shiftRight(128 - k.len + k.offset)
	if just.isZero() {
		ret = "0"
	} else {
		if just.hi > 0 {
			ret = fmt.Sprintf("%x", just.hi)
		}
		if just.lo > 0 {
			if just.hi > 0 {
				ret = fmt.Sprintf("%s%0*x", ret, (k.len-64)/4, just.lo)
			} else {
				ret = fmt.Sprintf("%s%x", ret, just.lo)
			}
		}
	}
	return fmt.Sprintf("%s/%d", ret, k.len-k.offset)
}

// truncated returns a copy of key truncated to n bits.
func (k key) truncated(n uint8) key {
	return newKey(k.content, k.offset, n)
}

// rest returns a copy of b starting at position i. if i > k.len, returns the
// zero key.
func (k key) rest(i uint8) key {
	if k.isZero() {
		return k
	}
	if i > k.len {
		i = 0
	}
	return newKey(k.content, i, k.len)
}

// hasBitZeroAt returns true if the bit at position i is 0.
// If i >= k.len, hasBitZeroAt returns false, false.
func (k key) hasBitZeroAt(i uint8) (isZero bool, ok bool) {
	if i >= k.len {
		return false, false
	}
	return k.content.isBitZero(i), true
}

// equalFromRoot reports whether k and o have the same content and len.
func (k key) equalFromRoot(o key) bool {
	return k.len == o.len && k.content == o.content
}

// commonPrefixLen returns the length of the common prefix between k and
// o, truncated to the length of the shorter of the two.
func (k key) commonPrefixLen(o key) uint8 {
	common := k.content.commonPrefixLen(o.content)
	// min(l.len, o.len, common)
	if k.len < o.len {
		if k.len < common {
			return k.len
		}
		return common
	} else {
		if o.len < common {
			return o.len
		}
		return common
	}
}

// isPrefixOf reports whether k has the same content as o up to position k.len.
func (k key) isPrefixOf(o key) bool {
	return k.len <= o.len && k.content == o.content.bitsClearedFrom(k.len)
}

// isZero reports whether k is the zero key.
func (k key) isZero() bool {
	// Bits beyond len are always ignored, so if k.len == zero, then this
	// key effectively contains no bits.
	return k.len == 0
}

// isValid reports whether k is a valid key.
// TODO: remove if not used
func (k key) isValid() bool {
	return k.offset < 128 && k.len <= 128
}

func (k key) left() key {
	return key{
		content: k.content,
		offset:  k.len,
		len:     k.len + 1,
	}
}
func (k key) right() key {
	return key{
		content: k.content.or(uint128{0, 1}.shiftLeft(128 - k.len - 1)),
		offset:  k.len,
		len:     k.len + 1,
	}
}
