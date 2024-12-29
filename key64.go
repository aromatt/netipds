package netipds

type key64 struct {
	content uint64
	offset  uint8
	len     uint8
}

// bitsSetFrom returns a copy of u with the given bit and all subsequent ones
// set.
func bitsSetFrom(u uint64, bit uint8) uint64 {
	return ^(u | mask64[bit])
}

// bitsClearedFrom returns a copy of u with the given bit and all subsequent
// ones cleared.
func bitsClearedFrom(u uint64, bit uint8) uint64 {
	return u & mask64[bit]
}
