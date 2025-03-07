package netipds

import (
	"fmt"
	"net/netip"
	"strings"
)

// key4 is an implementation of Key for 32-bit keys (IPv4).
type key4 struct {
	len     uint8
	offset  uint8
	content uint32
}

func (k key4) Offset() uint8 {
	return k.offset
}

func (k key4) Len() uint8 {
	return k.len
}

func (k key4) WithOffset(o uint8) key4 {
	return key4{k.len, o, k.content}
}

func bitsClearedFrom32(u uint32, bit uint8) uint32 {
	return u >> (32 - bit) << (32 - bit)
}

func NewKey4(content uint32, offset uint8, len uint8) key4 {
	return key4{len, offset, bitsClearedFrom32(content, len)}
}

// Rooted returns a copy of h with offset set to 0
func (h key4) Rooted() key4 {
	return key4{h.len, 0, h.content}
}

// ToPrefix returns the Prefix represented by k.
//func (k key4) ToPrefix() netip.Prefix {
//	var a4 [4]byte
//	bePutUint32(a4[:], k.content)
//	addr := netip.AddrFrom4(a4)
//	bits := int(k.len)
//	return netip.PrefixFrom(addr.Unmap(), bits)
//}

// key4FromPrefix returns the key that represents the provided Prefix.
func key4FromPrefix(p netip.Prefix) key4 {
	a4 := p.Addr().As4()
	return NewKey4(beUint32(a4[:]), 0, uint8(p.Bits()))
}

// String prints the key4's content in hex, followed by ",<offset>-<len>".
// The least significant bit in the output is the bit at position (h.len - 1).
// Leading zeros are omitted.
func (h key4) String() string {
	var content string
	// TODO remove
	just := h.content >> (32 - h.len)
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

	u32 := uint32(0)
	loStart := 0
	if _, err = fmt.Sscanf(contentStr[loStart:], "%x", &u32); err != nil {
		return fmt.Errorf("failed to parse key4: '%s', %w", h, err)
	}
	h.content = u32 << (32 - h.len)
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
	just := (h.content << h.offset) >> (32 - h.len + h.offset)
	if just == 0 {
		content = "0"
	} else {
		content = fmt.Sprintf("%x", just)
	}
	return fmt.Sprintf("%s,%d-%d", content, h.offset, h.len)
}

// Truncated returns a copy of key4 truncated to n bits.
func (h key4) Truncated(n uint8) key4 {
	return NewKey4(h.content, h.offset, n)
}

// Rest returns a copy of h starting at position i.
//
// Returns the zero key4 if i > h.len or h.isZero().
func (h key4) Rest(i uint8) key4 {
	if h.IsZero() || i > h.len {
		return key4{}
	}
	return NewKey4(h.content, i, h.len)
}

func isBitSet32(u uint32, bit uint8) uint8 {
	return uint8(u >> (31 - bit) & 1)
}

func (h key4) Bit(i uint8) bit {
	return bit(isBitSet32(h.content, i))
}

// EqualFromRoot reports whether h and o have the same content and len (offsets
// are ignored).
func (h key4) EqualFromRoot(o key4) bool {
	return h.len == o.len && h.content == o.content
}

// CommonPrefixLen returns the length of the common prefix between h and o,
// truncated to the length of the shorter of the two.
func (h key4) CommonPrefixLen(o key4) (n uint8) {
	return min(min(o.len, h.len), u32CommonPrefixLen(h.content, o.content))
}

// IsPrefixOf reports whether h has the same content as o up to position h.len.
//
// If strict, returns false if h == o.
func (h key4) IsPrefixOf(o key4, strict bool) bool {
	if h.len <= o.len && h.content == bitsClearedFrom32(o.content, h.len) {
		return !(strict && h.EqualFromRoot(o))
	}
	return false
}

// isZero reports whether h is the zero key4.
func (h key4) IsZero() bool {
	// Bits beyond len are always ignored, so if h.len == zero, then this
	// key4 effectively contains no bits.
	return h.len == 0
}

// next returns a one-bit key4 just beyond s, set to 1 if b == bitR.
// TODO should/does this handle partition-crossing?
func (h key4) Next(b bit) (ret key4) {
	switch b {
	case bitL:
		ret = key4{
			content: h.content,
			offset:  h.len,
			len:     h.len + 1,
		}
	case bitR:
		ret = key4{
			content: h.content | (uint32(1) << (32 - h.len - 1)),
			offset:  h.len,
			len:     h.len + 1,
		}
	}
	return
}

func (k key4) PathNext(path key4) bit {
	return path.Bit(k.len)
}
