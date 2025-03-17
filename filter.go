package netipds

// filter is a simple Bloom-like filter for uint128 keys.
type filter struct {
	// ones is simply all keys ORed together.
	ones uint128
	// zeros is the OR of all keys' inverses. It contains a 1 in every position
	// where no key has a 1.
	zeros uint128
	// minLen is the minimum length of keys that have been inserted.
	minLen uint8
	// maxLen is the maximum length of keys that have been inserted.
	maxLen uint8
}

// insert adds k to the filter.
func (f *filter) insert(k key[uint128]) {
	f.ones = f.ones.or(k.content)
	f.zeros = f.zeros.or(k.content.not())
	if f.minLen == 0 || k.len < f.minLen {
		f.minLen = k.len
	}
	if k.len > f.maxLen {
		f.maxLen = k.len
	}
}

// mightContain returns true if the filter might contain k.
func (f *filter) mightContain(k key[uint128]) bool {
	if k.len < f.minLen || k.len > f.maxLen {
		return false
	}
	if f.ones.and(k.content) != k.content {
		return false
	}
	notk := k.content.not()
	if f.zeros.and(notk) != notk {
		return false
	}
	return true
}

// mightContainPrefix returns true if the filter might contain a key that is a
// prefix of k.
func (f *filter) mightContainPrefix(k key[uint128]) bool {
	com1 := f.ones.and(k.content).commonPrefixLen(k.content)
	notk := k.content.not()
	com0 := f.zeros.and(notk).commonPrefixLen(notk)
	return com1 >= f.minLen && com0 >= f.minLen
}
