package netipmap

import (
	"fmt"
	"net/netip"
	"strings"
)

// label represents one node's key fragment. The content of the label occupies
// the most-significant len bits of the value field. value should not have any
// bits set beyond len (using newLabel() enforces this).
type label struct {
	value uint128
	len   uint8
}

func newLabel(value uint128, len uint8) label {
	return label{value: value.bitsClearedFrom(len), len: len}
}

func labelFromPrefix(prefix netip.Prefix) label {
	return newLabel(u128From16(prefix.Addr().As16()), uint8(prefix.Bits()))
}

// Prints l.value as hex, followed by "/len". The least significant bit in the
// output is the bit at position l.len. Leading zeros are omitted. Examples:
//
//   - label{uint128{0, 1}, 128} => "1/128"
//   - label{uint128{0, 2}, 128} => "2/128"
//   - label{uint128{0, 2}, 127} => "1/127"
//   - label{uint128{1, 1}, 128} => "10000000000000001/128"
//   - label{uint128{1, 0}, 64}  => "1/64"
//   - label{uint128{256, 0}, 56} => "1/56"
//   - label{uint128{256, 0}, 64} => "100/64"
func (l label) String() string {
	var ret string
	just := l.value.shiftRight(128 - l.len)
	if l.value.hi == 0 && l.value.lo == 0 {
		ret = "0"
	} else {
		if just.hi > 0 {
			ret = fmt.Sprintf("%x", just.hi)
		}
		if just.lo > 0 {
			if just.hi > 0 {
				ret = fmt.Sprintf("%s%0*x", ret, (l.len-64)/4, just.lo)
			} else {
				ret = fmt.Sprintf("%s%x", ret, just.lo)
			}
		}
	}
	return fmt.Sprintf("%s/%d", ret, l.len)
}

// Parse() is the inverse of String().
func (l *label) Parse(s string) error {
	var err error

	// Isolate value and len
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse label '%s': invalid format", s)
	}
	valueStr, lenStr := parts[0], parts[1]
	if _, err = fmt.Sscanf(lenStr, "%d", &l.len); err != nil {
		return fmt.Errorf("failed to parse label '%s': %w", s, err)
	}

	// lo = right-most 64 bits
	// hi = anything to the left of lo
	hi, lo := uint64(0), uint64(0)
	loStart := 0
	if len(valueStr) > 16 {
		loStart = len(valueStr) - 16
		if _, err = fmt.Sscanf(valueStr[:loStart], "%x", &hi); err != nil {
			return fmt.Errorf("failed to parse label: '%s', %w", s, err)
		}
	}
	if _, err = fmt.Sscanf(valueStr[loStart:], "%x", &lo); err != nil {
		return fmt.Errorf("failed to parse label: '%s', %w", s, err)
	}
	l.value = uint128{hi, lo}.shiftLeft(128 - l.len)
	return nil
}

// bitsClearedFrom returns a copy of label truncated to n bits.
func (l label) truncated(n uint8) label {
	return newLabel(l.value, n)
}

// rest returns a copy of l starting at the bit at position i.
// if i > l.len, returns the zero label.
func (l label) rest(i uint8) label {
	if l.isZero() {
		return l
	}
	if i > l.len {
		i = 0
	}
	return newLabel(l.value.shiftLeft(i), l.len-i)
}

// isBitZero returns the value of the bit at position i.
// If i >= l.len, isBitZero returns false, false.
func (l label) isBitZero(i uint8) (isZero bool, ok bool) {
	if i >= l.len {
		return false, false
	}
	return l.value.isBitZero(i), true
}

// concat returns a new label with the bits of m appended to l.
func (l label) concat(m label) label {
	newLen := l.len + m.len
	if newLen > 128 {
		newLen = 128
	}
	return newLabel(l.value.or(m.value.shiftRight(l.len)), newLen)
}

// commonPrefixLen returns the length of the common prefix between l and
// m, truncated to the length of the shorter of the two.
func (l label) commonPrefixLen(m label) uint8 {
	common := l.value.commonPrefixLen(m.value)
	// min(l.len, m.len, common)
	if l.len < m.len {
		if l.len < common {
			return l.len
		}
		return common
	} else {
		if m.len < common {
			return m.len
		}
		return common
	}
}

func (l label) isPrefixOf(m label) bool {
	return l.len <= m.len && l.value == m.value.bitsClearedFrom(l.len)
}

func (l label) isZero() bool {
	return l == label{}
}

// TODO remove if not used
func (l label) isValid() bool {
	return l.len <= 128
}

// If the shorter of l and m is a prefix of the longer, return the length of
// the longer label. Otherwise, return the length of the common prefix,
// truncated to the length of the shorter label.
func (l label) prefixUnionLen(m label) uint8 {
	if l.len == m.len {
		return l.commonPrefixLen(m)
	} else {
		var shorter, longer label
		if l.len < m.len {
			shorter, longer = l, m
		} else {
			shorter, longer = m, l
		}
		if shorter.isPrefixOf(longer) {
			return longer.len
		}
		return shorter.commonPrefixLen(longer)
	}
}
