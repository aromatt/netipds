package netipds

import (
	"net/netip"
	"testing"
)

func TestPrefixSetAddContains(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), false},
		{pfxs("::0/127"), pfx("::0/128"), false},
		{pfxs("::0/127", "::0/128"), pfx("::0/128"), true},
		{pfxs("::0/127", "::1/128"), pfx("::1/128"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("9.9.9.0/24"), false},

		// encompassed, but not contained
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},
		{pfxs("0.0.0.0/1", "128.0.0.0/1"), pfx("128.0.0.0/1"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},

		// exercises tree.newParent
		{
			pfxs("128.0.0.0/32", "64.0.0.0/32", "32.0.0.0/32", "16.0.0.0/32"),
			pfx("16.0.0.0/32"),
			true,
		},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.4/32"), pfx("::ffff:1.2.3.4/128"), false},
		{pfxs("1.2.3.4/32"), pfx("1.2.3.4/32"), true},
		{pfxs("::ffff:1.2.3.4/128"), pfx("1.2.3.4/32"), false},
		{pfxs("::ffff:1.2.3.4/128"), pfx("::ffff:1.2.3.4/128"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Contains(tt.get); got != tt.want {
			t.Errorf("ps.Contains(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixSetEncompasses(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), false},
		{pfxs("::0/127"), pfx("::0/128"), true},
		// The set covers the input prefix but does not encompass it.
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), true},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Encompasses(tt.get); got != tt.want {
			t.Errorf("ps.Encompasses(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixSetRootOf(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},

		// RootOf will return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), pfx("::0/128"), true},

		// Make sure entry-less nodes are not returned by RootOf
		{pfxs("::0/127", "::2/127"), pfx("::0/128"), pfx("::0/127"), true},

		// IPv4
		{pfxs(), pfx("1.2.3.0/32"), netip.Prefix{}, false},
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		gotPrefix, gotOK := ps.RootOf(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"ps.RootOf(%s) = (%v, _, %v), want (%v, _, %v)",
				tt.get, gotPrefix, gotOK, tt.wantPrefix, tt.wantOK,
			)
		}
	}
}

func TestPrefixSetParentOf(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},

		// ParentOf will return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), pfx("::0/128"), true},

		// IPv4
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), pfx("1.2.3.0/32"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		gotPrefix, gotOK := ps.ParentOf(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"ps.ParentOf(%s) = (%v, _, %v), want (%v, _, %v)",
				tt.get, gotPrefix, gotOK, tt.wantPrefix, tt.wantOK,
			)
		}
	}
}

func TestPrefixSetDescendantsOf(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfx("::0/128"), pfxs()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), pfxs()},
		{pfxs("::1/128"), pfx("::0/128"), pfxs()},
		{pfxs("::0/128"), pfx("::0/128"), pfxs("::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), pfxs("::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), pfxs("::2/128")},
		{pfxs("::0/128"), pfx("::1/127"), pfxs("::0/128")},
		{pfxs("::1/128"), pfx("::0/127"), pfxs("::1/128")},
		{pfxs("::2/127"), pfx("::2/127"), pfxs("::2/127")},

		// Using "::/0" as a lookup key
		{pfxs("::0/128"), pfx("::/0"), pfxs("::0/128")},

		// Get a prefix that has no entry but has children.
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: pfxs("::0/128", "::1/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128", "::2/128"),
			get:  pfx("::2/127"),
			want: pfxs("::2/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: pfxs("::0/128", "::1/128"),
		},
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::2/127"),
			want: pfxs("::2/128", "::3/128"),
		},

		// Get an entry-less shared prefix node that has an entry-less child
		{
			set: pfxs("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// entry-less shared prefixes.
			get:  pfx("::4/126"),
			want: pfxs("::4/128", "::6/128", "::7/128"),
		},

		// Get a node that is both an entry and a shared prefix node and has an
		// entry-less child
		{
			set: pfxs("::4/126", "::6/128", "::7/128"),
			get: pfx("::4/126"),
			// The node "::6/127" is a node in the tree but has no entry, so it
			// should not be included in the result.
			want: pfxs("::4/126", "::6/128", "::7/128"),
		},

		// Get a prefix that has no exact node, but still has descendants
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::0/126"),
			want: pfxs("::2/128", "::3/128"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), pfxs("1.2.3.0/32")},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/24"), pfxs("1.2.3.0/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.3.0/24"), pfxs("1.2.3.1/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.4.0/24"), pfxs()},
		{
			set:  pfxs("1.2.3.0/32", "1.2.3.1/32"),
			get:  pfx("1.2.3.0/24"),
			want: pfxs("1.2.3.0/32", "1.2.3.1/32"),
		},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		checkPrefixSlice(t, psb.PrefixSet().DescendantsOf(tt.get).Prefixes(), tt.want)
	}
}

func TestPrefixSetAncestorsOf(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfx("::0/128"), pfxs()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), pfxs()},
		{pfxs("::1/128"), pfx("::0/128"), pfxs()},
		{pfxs("::0/128"), pfx("::0/128"), pfxs("::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), pfxs("::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), pfxs("::2/128")},
		{pfxs("::0/127"), pfx("::0/128"), pfxs("::0/127")},
		{pfxs("::0/127"), pfx("::1/128"), pfxs("::0/127")},
		{pfxs("::2/127"), pfx("::2/127"), pfxs("::2/127")},

		// Multi-prefix maps
		{
			set:  pfxs("::0/127", "::0/128"),
			get:  pfx("::0/128"),
			want: pfxs("::0/127", "::0/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/128"),
			want: pfxs("::0/128"),
		},
		{
			set:  pfxs("::0/126", "::0/127", "::1/128"),
			get:  pfx("::0/128"),
			want: pfxs("::0/126", "::0/127"),
		},

		// Make sure nodes without entries are excluded
		{
			set: pfxs("::0/128", "::2/128"),
			get: pfx("::0/128"),
			// "::2/127" is a node in the tree but has no entry, so it should
			// not be included in the result.
			want: pfxs("::0/128"),
		},

		// Make sure parent/child insertion order doesn't matter
		{
			set:  pfxs("::0/126", "::0/127"),
			get:  pfx("::0/128"),
			want: pfxs("::0/126", "::0/127"),
		},
		{
			set:  pfxs("::0/127", "::0/126"),
			get:  pfx("::0/128"),
			want: pfxs("::0/126", "::0/127"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.1/32"), pfxs()},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), pfxs("1.2.3.0/32")},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/32"), pfxs("1.2.3.0/24")},
		// Insert shortest prefix first
		{
			set:  pfxs("1.2.0.0/16", "1.2.3.0/24"),
			get:  pfx("1.2.3.0/32"),
			want: pfxs("1.2.0.0/16", "1.2.3.0/24"),
		},
		// Insert longest prefix first
		{
			set:  pfxs("1.2.3.0/24", "1.2.0.0/16"),
			get:  pfx("1.2.3.0/32"),
			want: pfxs("1.2.0.0/16", "1.2.3.0/24"),
		},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		checkPrefixSlice(t, psb.PrefixSet().AncestorsOf(tt.get).Prefixes(), tt.want)
	}
}

func TestPrefixSetOverlapsPrefix(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/127"), pfx("::0/128"), true},
		{pfxs("::0/128", "::1/128"), pfx("::2/128"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.0.0/16"), true},

		// Make sure value-less nodes don't count. This PrefixSet contains
		// the shared prefix ::0/126.
		{pfxs("::0/128", "::2/128"), pfx("::3/128"), false},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.OverlapsPrefix(tt.get); got != tt.want {
			t.Errorf("ps.OverlapsPrefix(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func checkPrefixSlice(t *testing.T, got, want []netip.Prefix) {
	if len(got) != len(want) {
		t.Errorf("got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
		return
	}
	for i, p := range got {
		if p != want[i] {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}

}

func TestPrefixSetSubtractPrefix(t *testing.T) {
	tests := []struct {
		set      []netip.Prefix
		subtract netip.Prefix
		want     []netip.Prefix
	}{
		{pfxs(), netip.Prefix{}, pfxs()},
		{pfxs("::0/1"), pfx("::0/1"), pfxs()},
		{pfxs("::0/2"), pfx("::0/2"), pfxs()},
		{pfxs("::0/128"), pfx("::0/128"), pfxs()},
		{pfxs("::0/128"), pfx("::0/127"), pfxs()},
		{pfxs("::0/128"), pfx("::1/128"), pfxs("::0/128")},
		{pfxs("::0/127"), pfx("::0/128"), pfxs("::1/128")},
		{pfxs("::2/127"), pfx("::3/128"), pfxs("::2/128")},
		{pfxs("::0/126"), pfx("::0/128"), pfxs("::1/128", "::2/127")},
		{pfxs("::0/126"), pfx("::3/128"), pfxs("::0/127", "::2/128")},

		// Subtract from empty set
		{pfxs(), netip.Prefix{}, pfxs()},
		{pfxs(), pfx("::0/1"), pfxs()},

		// IPv4
		{
			set:      pfxs("1.2.3.0/30"),
			subtract: pfx("1.2.3.0/32"),
			want:     pfxs("1.2.3.1/32", "1.2.3.2/31"),
		},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{
			set:      pfxs("1.2.3.0/30"),
			subtract: pfx("::ffff:1.2.3.0/128"),
			want:     pfxs("1.2.3.0/30"),
		},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		psb.SubtractPrefix(tt.subtract)
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSetSubtract(t *testing.T) {
	tests := []struct {
		set      []netip.Prefix
		subtract []netip.Prefix
		want     []netip.Prefix
	}{
		{pfxs("::0/1"), pfxs("::0/1"), pfxs()},
		{pfxs("::0/2"), pfxs("::0/2"), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128")},
		{pfxs("::0/127"), pfxs("::0/128"), pfxs("::1/128")},
		{pfxs("::2/127"), pfxs("::3/128"), pfxs("::2/128")},
		{pfxs("::0/126"), pfxs("::0/128"), pfxs("::1/128", "::2/127")},
		{pfxs("::0/126"), pfxs("::3/128"), pfxs("::0/127", "::2/128")},
		{pfxs("::0/127"), pfxs("::0/128", "::1/128"), pfxs()},
		{pfxs("::3/128"), pfxs("::2/127"), pfxs()},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128"), pfxs("::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128", "::1/128"), pfxs()},
		{pfxs("::0/127", "::1/128"), pfxs("::0/127"), pfxs()},
		{pfxs("::3/128"), pfxs("::2/127", "::1/128"), pfxs()},

		// Subtract from empty set
		{pfxs(), pfxs(), pfxs()},
		{pfxs(), pfxs("::0/1"), pfxs()},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{
			set:      pfxs("1.2.3.0/30"),
			subtract: pfxs("::ffff:1.2.3.0/128"),
			want:     pfxs("1.2.3.0/30"),
		},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		subPsb := &PrefixSetBuilder{}
		for _, p := range tt.subtract {
			subPsb.Add(p)
		}
		psb.Subtract(subPsb.PrefixSet())
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSetIntersect(t *testing.T) {
	tests := []struct {
		a    []netip.Prefix
		b    []netip.Prefix
		want []netip.Prefix
	}{
		// Note: since intersect is commutative, all test cases are performed
		// twice (a & b) and (b & a)
		{pfxs("::0/128"), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::2/127"), pfxs()},
		{pfxs("::0/128", "::1/128"), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/126"), pfxs("::0/128")},
		{pfxs("::1/128"), pfxs("::0/127"), pfxs("::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::1/128", "::4/126"), pfxs("::0/127"), pfxs("::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/127"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/126"), pfxs("::0/128", "::1/128")},
		{pfxs("::2/127"), pfxs("::0/126", "::2/128"), pfxs("::2/127", "::2/128")},
		{pfxs("::2/127"), pfxs("::0/126", "::0/128"), pfxs("::2/127")},
		{pfxs("::2/127", "::3/128"), pfxs("::0/126", "::0/128"), pfxs("::2/127", "::3/128")},

		// IPv4
		{pfxs("1.2.3.0/24"), pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.0/24"), pfxs("1.2.0.0/32"), pfxs()},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.0/24"), pfxs("::ffff:1.2.3.4/128"), pfxs()},
	}
	performTest := func(x, y []netip.Prefix, want []netip.Prefix) {
		psb := &PrefixSetBuilder{}
		for _, p := range x {
			psb.Add(p)
		}
		intersectPsb := &PrefixSetBuilder{}
		for _, p := range y {
			intersectPsb.Add(p)
		}
		psb.Intersect(intersectPsb.PrefixSet())
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), want)
	}

	for _, tt := range tests {
		performTest(tt.a, tt.b, tt.want)
		performTest(tt.b, tt.a, tt.want)
	}
}

func TestPrefixSetMerge(t *testing.T) {
	tests := []struct {
		a    []netip.Prefix
		b    []netip.Prefix
		want []netip.Prefix
	}{
		// Note: since union is commutative, all test cases are performed twice
		// (a | b) and (b | a)
		{pfxs(), pfxs(), pfxs()},
		{pfxs("::0/1"), pfxs(), pfxs("::0/1")},
		{pfxs("::0/1"), pfxs("::0/1"), pfxs("::0/1")},
		{pfxs("::0/2"), pfxs("::0/2"), pfxs("::0/2")},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/127", "::0/128")},
		{pfxs("::0/128", "::1/128"), pfxs(), pfxs("::0/128", "::1/128")},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128", "0::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128"), pfxs("::0/128", "0::1/128")},
		{
			pfxs("::0/127"),
			pfxs("::0/128", "::1/128"),
			pfxs("::0/127", "::0/128", "::1/128"),
		},
		{
			pfxs("::2/127"),
			pfxs("::0/126", "::2/128"),
			pfxs("::0/126", "::2/127", "::2/128"),
		},
		{
			pfxs("::0/128", "::1/128"),
			pfxs("::0/126", "::0/127"),
			pfxs("::0/126", "::0/127", "::0/128", "::1/128"),
		},
		{
			pfxs("::0/128", "::1/128"),
			pfxs("::0/126", "::0/127", "::2/127"),
			pfxs("::0/126", "::0/127", "::0/128", "::1/128", "::2/127"),
		},

		// IPv4
		{pfxs("1.2.3.4/32"), pfxs(), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.0/24"), pfxs("1.2.3.0/24", "1.2.3.4/32")},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{
			pfxs("1.2.3.4/32"),
			pfxs("::ffff:1.2.3.4/128"),
			pfxs("::ffff:1.2.3.4/128", "1.2.3.4/32"),
		},
	}
	performTest := func(x, y []netip.Prefix, want []netip.Prefix) {
		psb := &PrefixSetBuilder{}
		for _, p := range x {
			psb.Add(p)
		}
		unionPsb := &PrefixSetBuilder{}
		for _, p := range y {
			unionPsb.Add(p)
		}
		psb.Merge(unionPsb.PrefixSet())
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), want)
	}
	for _, tt := range tests {
		performTest(tt.a, tt.b, tt.want)
		performTest(tt.b, tt.a, tt.want)
	}
}

/* HACK
func TestPrefixSetRemove(t *testing.T) {
	tests := []struct {
		add    []netip.Prefix
		remove []netip.Prefix
		want   []netip.Prefix
	}{
		{pfxs(), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs(), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/128")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		for _, p := range tt.remove {
			psb.Remove(p)
		}
		ps := psb.PrefixSet()
		checkPrefixSlice(t, ps.Prefixes(), tt.want)
	}
}

func TestPrefixSetFilter(t *testing.T) {
	tests := []struct {
		add    []netip.Prefix
		filter []netip.Prefix
		want   []netip.Prefix
	}{
		{pfxs(), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/128")},
		{pfxs("::0/127"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/127"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/126"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/128", "::2/128"), pfxs("::0/127"), pfxs("::0/128")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		filterPsb := &PrefixSetBuilder{}
		for _, p := range tt.filter {
			filterPsb.Add(p)
		}
		psb.Filter(filterPsb.PrefixSet())
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}
*/

func TestPrefixSetPrefixesCompact(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/127", "::0/128"), pfxs("::0/127")},
		{pfxs("::0/126", "::0/127"), pfxs("::0/126")},
		{pfxs("::0/1", "::0/128"), pfxs("::0/1")},
		{pfxs("8000::/1"), pfxs("8000::/1")},
		{pfxs("::0/1", "8000::/1"), pfxs("::0/1", "8000::/1")},
		{pfxs("0::0/127", "::0/128", "::1/128"), pfxs("::0/127")},
		{pfxs("0::0/127", "::0/128", "::2/128"), pfxs("::0/127", "::2/128")},

		// IPv4
		{pfxs("1.2.3.0/24", "1.2.3.4/32"), pfxs("1.2.3.0/24")},
		{pfxs("1.2.3.0/31", "1.2.3.2/32"), pfxs("1.2.3.0/31", "1.2.3.2/32")},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("::ffff:1.2.3.4/128", "1.2.3.4/32"), pfxs("::ffff:1.2.3.4/128", "1.2.3.4/32")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		checkPrefixSlice(t, ps.PrefixesCompact(), tt.want)
	}
}

func TestPrefixSetSize(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want int
	}{
		{pfxs(), 0},
		{pfxs("::0/128"), 1},
		{pfxs("8000::/1"), 1},
		{pfxs("::0/128", "::0/128"), 1},
		{pfxs("::0/128", "::1/128"), 2},
		{pfxs("::0/128", "8000::/1"), 2},
		{pfxs("::0/127", "::0/128"), 2},
		{pfxs("::0/126", "::0/127"), 2},
		{pfxs("::0/127", "::0/128", "::1/128"), 3},
		{pfxs("::0/128", "1.2.3.4/32"), 2},
		{pfxs("::0/128", "::0/128"), 1},
		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.4/32", "::ffff:1.2.3.4/128"), 2},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Size(); got != tt.want {
			t.Errorf("ps.Size() = %d, want %d", got, tt.want)
		}
	}
}
