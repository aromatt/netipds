package netipds

// mustPrefixOf panics unless a is a prefix of b.
//
// Callers use this to guard a descent that is only meaningful while the node
// being visited sits at or above the key being removed. A violation means the
// tree is malformed, so there is nothing sensible to return.
func mustPrefixOf[B keybits[B]](a, b key[B]) {
	if !a.IsPrefixOf(b) {
		panic("netipds: malformed tree: " + a.String() +
			" is not a prefix of " + b.String())
	}
}
