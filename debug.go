//go:build debug

package netipds

import (
	"fmt"
)

// DumpTree prints a human-readable representation of s's internal tree structure.
func (m *PrefixMapBuilder[T]) DumpTree() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		m.tree4.stringImpl("", "", false),
		m.tree6.stringImpl("", "", false),
	)
}

// DumpTree prints a human-readable representation of s's internal tree structure.
func (m *PrefixMap[T]) DumpTree() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		m.tree4.stringImpl("", "", false),
		m.tree6.stringImpl("", "", false),
	)
}

// DumpTree prints a human-readable representation of s's internal tree structure.
func (s *PrefixSetBuilder) DumpTree() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", true),
		s.tree6.stringImpl("", "", true),
	)
}

// DumpTree prints a human-readable representation of s's internal tree structure.
func (s *PrefixSet) DumpTree() string {
	return fmt.Sprintf("IPv4:\n%s\nIPv6:\n%s",
		s.tree4.stringImpl("", "", true),
		s.tree6.stringImpl("", "", true),
	)
}

func (t *tree[T, B]) stringImpl(indent string, pre string, hideVal bool) string {
	var ret string
	var valstr string
	if t.hasEntry {
		valstr = ": "
		if hideVal {
			valstr += "(*)"
		} else {
			valstr += fmt.Sprintf("%v", t.value)
		}
	}
	ret = fmt.Sprintf("%s%s%s%s\n", indent, pre, t.key.String(), valstr)
	if t.left != nil {
		ret += t.left.stringImpl(indent+"  ", "L:", hideVal)
	}
	if t.right != nil {
		ret += t.right.stringImpl(indent+"  ", "R:", hideVal)
	}
	return ret
}

func (t *tree[T, B]) String() string {
	return t.stringImpl("", "", false)
}
