package netipds

//import (
//	"net/netip"
//)

// Key stores the string of bits which represent the full path to a node in a
// prefix tree.
type Key[K any] interface {
	// Offset returns the starting position of the bit range owned by this key.
	Offset() uint8

	WithOffset(uint8) K

	// Len returns the ending position of the bit range owned by this key.
	Len() uint8

	// Rooted returns a copy of key with offset set to 0.
	Rooted() K

	// ToPrefix returns the Prefix represented by the key.
	//ToPrefix() netip.Prefix

	// String prints the key's content in hex, followed by "," + k.len. The least
	// significant bit in the output is the bit at position (k.len - 1). Leading
	// zeros are omitted.
	String() string

	// TODO
	StringRel() string

	// Truncated returns a copy of key truncated to n bits.
	Truncated(uint8) K

	// Rest returns a copy of the key starting at position i. if i > k.len,
	// returns the zero key.
	Rest(i uint8) K

	// Bit returns the value (as a `bit`) of the big at the provided offset.
	Bit(uint8) bit

	// EqualFromRoot reports whether the key and o have the same content and
	// len (offsets are ignored).
	EqualFromRoot(o K) bool

	// CommonPrefixLen returns the length of the common prefix between the key
	// and o, truncated to the length of the shorter of the two.
	CommonPrefixLen(o K) uint8

	// IsPrefixOf reports whether the key has the same content as o up to
	// position k.len.
	//
	// If strict, returns false if the key == o.
	IsPrefixOf(o K, strict bool) bool

	// IsZero reports whether k is the zero key.
	IsZero() bool

	// Next returns a one-bit key just beyond the key's len, set to 1 if b ==
	// bitR.
	Next(b bit) K
}
