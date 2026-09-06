package netipds

import (
	"net/netip"
	"testing"
)

func TestPrefixSetInvalidPrefix(t *testing.T) {
	psb := &PrefixSetBuilder{}
	invalidPrefix := netip.Prefix{}

	if psb.Add(invalidPrefix) == nil {
		t.Errorf("Expected err != nil")
	}

	if psb.Remove(invalidPrefix) == nil {
		t.Errorf("Expected err != nil")
	}

	if psb.SubtractPrefix(invalidPrefix) == nil {
		t.Errorf("Expected err != nil")
	}

	ps := psb.PrefixSet()

	if ps.Size() != 0 {
		t.Errorf("Expected ps.Size() to be false")
	}

	if ps.Contains(invalidPrefix) {
		t.Errorf("Expected ps.Contains(%s) to be false", invalidPrefix)
	}

	if ps.Encompasses(invalidPrefix) {
		t.Errorf("Expected ps.Encompasses(%s) to be false", invalidPrefix)
	}

	if ps.OverlapsPrefix(invalidPrefix) {
		t.Errorf("Expected ps.OverlapsPrefix(%s) to be false", invalidPrefix)
	}

	if _, ok := ps.RootOf(invalidPrefix); ok {
		t.Errorf("Expected ps.RootOf(%s) to return ok=false", invalidPrefix)
	}

	if _, ok := ps.ParentOf(invalidPrefix); ok {
		t.Errorf("Expected ps.ParentOf(%s) to return ok=false", invalidPrefix)
	}

	if ps.Subnets(invalidPrefix).Size() != 0 {
		t.Errorf("Expected ps.Subnets(%s) to return empty slice", invalidPrefix)
	}

	if ps.Supernets(invalidPrefix).Size() != 0 {
		t.Errorf("Expected ps.Supernets(%s) to return empty slice", invalidPrefix)
	}
}

func TestPrefixSetContains(t *testing.T) {
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), true},
		{pfxs("::0/0"), pfx("::1/128"), true},
		{pfxs("0.0.0.0/0"), pfx("1.2.3.4/32"), true},

		// The set covers the input prefix but does not encompass it.
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), true},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), pfx("0.0.0.0/0"), true},
		{pfxs("::0/0"), pfx("::1/128"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("1.2.3.4/32"), pfx("0.0.0.0/0"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), pfx("0.0.0.0/0"), true},
		{pfxs("::0/0"), pfx("::1/128"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("1.2.3.4/32"), pfx("0.0.0.0/0"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
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

func TestPrefixSetSubnets(t *testing.T) {
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), pfxs("::0/0")},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), pfxs("0.0.0.0/0")},
		{pfxs("::1/128"), pfx("::0/0"), pfxs("::1/128")},
		{pfxs("1.2.3.4/32"), pfx("0.0.0.0/0"), pfxs("1.2.3.4/32")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
		}
		assertEqualPrefixSlice(t, psb.PrefixSet().Subnets(tt.get).Prefixes(), tt.want)
	}
}

func TestPrefixSetSupernets(t *testing.T) {
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

		// Default routes
		{pfxs("::0/0"), pfx("::0/0"), pfxs("::0/0")},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), pfxs("0.0.0.0/0")},
		{pfxs("::0/0"), pfx("::1/128"), pfxs("::0/0")},
		{pfxs("0.0.0.0/0"), pfx("1.2.3.4/32"), pfxs("0.0.0.0/0")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
		}
		assertEqualPrefixSlice(t, psb.PrefixSet().Supernets(tt.get).Prefixes(), tt.want)
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

		// Default routes overlap with themselves
		{pfxs("::0/0"), pfx("::0/0"), true},
		{pfxs("0.0.0.0/0"), pfx("0.0.0.0/0"), true},
		{pfxs("::0/0"), pfx("1234::5678/128"), true},
		{pfxs("0.0.0.0/0"), pfx("1.2.3.4/32"), true},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
		}
		ps := psb.PrefixSet()
		if got := ps.OverlapsPrefix(tt.get); got != tt.want {
			t.Errorf("ps.OverlapsPrefix(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

// assertEqualPrefixSlice asserts that the two slices contain the same Prefixes
// TODO just use slices.Equal
func assertEqualPrefixSlice(t *testing.T, got, want []netip.Prefix) {
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

var subtractPrefixTests = []struct {
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

	// Subtract from empty set
	{pfxs(), pfxs("::0/1"), pfxs()},

	// IPv4
	{
		set:      pfxs("1.2.3.0/30"),
		subtract: pfxs("1.2.3.0/32"),
		want:     pfxs("1.2.3.1/32", "1.2.3.2/31"),
	},

	// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
	{
		set:      pfxs("1.2.3.0/30"),
		subtract: pfxs("::ffff:1.2.3.0/128"),
		want:     pfxs("1.2.3.0/30"),
	},

	// Default routes - themselves
	{pfxs("::0/0"), pfxs(), pfxs("::0/0")},
	{pfxs("0.0.0.0/0"), pfxs(), pfxs("0.0.0.0/0")},
	{pfxs("::0/0"), pfxs("::0/0"), pfxs()},
	{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0"), pfxs()},

	// Default routes - subsets of themselves
	{pfxs("::0/0"), pfxs("::0/2"), pfxs("4000::0/2", "8000::/1")},
	{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/2"), pfxs("64.0.0.0/2", "128.0.0.0/1")},
}

func TestPrefixSetBuilderSubtractPrefix(t *testing.T) {
	for _, tt := range subtractPrefixTests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
		}
		for _, p := range tt.subtract {
			tErr(psb.SubtractPrefix(p), t)
		}
		assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSet1MPrefixes(t *testing.T) {
	// Create a PrefixSet containing 1 million prefixes spread across the IPv4
	// space, and then verify that it contains exactly 1 million prefixes.
	psb := &PrefixSetBuilder{}
	var a4 [4]byte
	for i := 0; i < 1_000_000; i++ {
		bePutUint32(a4[:], uint32(i*100_000+i))
		a := netip.AddrFrom4(a4)
		p := netip.PrefixFrom(a, 32)
		tErr(psb.Add(p), t)
	}
	ps := psb.PrefixSet()
	if len(ps.Prefixes()) != 1_000_000 {
		t.Errorf("got %d prefixes, want 1000000", len(ps.Prefixes()))
	}
}

func TestPrefixSetBuilderRemoveDefaultRoute(t *testing.T) {
	// Test removing default routes specifically to catch tree structure bugs
	tests := []struct {
		name   string
		prefix netip.Prefix
		after  []netip.Prefix
	}{
		{"IPv4 default", pfx("0.0.0.0/0"), pfxs("0.0.0.0/32", "0.0.0.1/32")},
		{"IPv6 default", pfx("::0/0"), pfxs("::0/128", "::1/128")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add only the default route
			psb := &PrefixSetBuilder{}
			tErr(psb.Add(tt.prefix), t)

			// Verify it was added
			ps := psb.PrefixSet()
			if !ps.Contains(tt.prefix) {
				t.Errorf("set should contain %v after adding", tt.prefix)
			}
			if ps.Size() != 1 {
				t.Errorf("set size should be 1, got %d", ps.Size())
			}

			// Remove the default route
			tErr(psb.Remove(tt.prefix), t)

			// Verify it was removed and set is empty
			ps = psb.PrefixSet()
			if ps.Contains(tt.prefix) {
				t.Errorf("set should not contain %v after removal", tt.prefix)
			}
			if ps.Size() != 0 {
				t.Errorf("set size should be 0 after removal, got %d", ps.Size())
			}
			if len(ps.Prefixes()) != 0 {
				t.Errorf("Prefixes() should return empty slice, got %v", ps.Prefixes())
			}

			// Add prefixes to empty set and verify
			// (This tests the use of a PrefixSetBuilder after removing the default route)
			for _, p := range tt.after {
				tErr(psb.Add(p), t)
			}
			ps = psb.PrefixSet()
			if len(ps.Prefixes()) != len(tt.after) {
				t.Errorf("after adding prefixes, got %d prefixes, want %d",
					len(ps.Prefixes()), len(tt.after))
			}
			for _, p := range tt.after {
				if !ps.Contains(p) {
					t.Errorf("after adding, set should contain %v", p)
				}
			}
		})
	}
}

var subtractTests = []struct {
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

	// This test covers https://github.com/aromatt/netipds/issues/31
	{pfxs("::0/128"), pfxs("::0/128", "::1/128"), pfxs()},

	// Subtract from empty set
	{pfxs(), pfxs(), pfxs()},
	{pfxs(), pfxs("::0/1"), pfxs()},

	// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
	{
		set:      pfxs("1.2.3.0/30"),
		subtract: pfxs("::ffff:1.2.3.0/128"),
		want:     pfxs("1.2.3.0/30"),
	},

	// Default routes
	{pfxs("::0/0"), pfxs("::0/0"), pfxs()},
	{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0"), pfxs()},

	// Subtract a non-overlapping prefix
	{pfxs("128.0.0.0/1"), pfxs("0.0.0.0/2"), pfxs("128.0.0.0/1")},

	// Previously failing example discovered by property-based test. This test case
	// failed to match the netipx implementation.
	{
		set:      pfxs("64.0.0.0/3", "0.0.0.0/3"), // 010, 000
		subtract: pfxs("0.0.0.0/2"),               // 00
		want:     pfxs("64.0.0.0/3"),              // 010
	},

	// Previously failing example discovered by property-based test. This test case failed
	// to match the netipx implementation.
	//
	// In this example, we start with a single node, then subtract both
	// grandchildren on one side, leaving only the immediate child on the other
	// side.
	{
		set:      pfxs("0.0.0.0/2"),               // 00
		subtract: pfxs("0.0.0.0/4", "16.0.0.0/4"), // 0000, 0001
		want:     pfxs("32.0.0.0/3"),              // 001
	},
}

func TestPrefixSetBuilderSubtract(t *testing.T) {
	for _, tt := range subtractTests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			tErr(psb.Add(p), t)
		}
		subPsb := &PrefixSetBuilder{}
		for _, p := range tt.subtract {
			tErr(subPsb.Add(p), t)
		}
		psb.Subtract(subPsb.PrefixSet())
		psSubtracted := psb.PrefixSet()
		assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)

		// Compare result against netipx.IPSet implementation
		ipset, err := ipsetSubtract(psb.PrefixSet(), subPsb.PrefixSet())
		if err != nil {
			t.Fatalf("Oracle IPSet build failed: %v", err)
		}
		netipdsIpSet := prefixSetToIPset(psSubtracted)
		if !netipdsIpSet.Equal(ipset) {
			t.Errorf("IP space mismatch against netipx implementation:\nA: %v\nB: %v\nExpected: %v\nActual: %v",
				tt.set, tt.subtract, ipset.Prefixes(), netipdsIpSet.Prefixes())
		}
	}
}

func TestPrefixSetBuilderSubtractPreservesRoot(t *testing.T) {
	psb := &PrefixSetBuilder{}
	tErr(psb.Add(pfx("0.0.0.0/0")), t)

	sub := &PrefixSetBuilder{}
	tErr(sub.Add(pfx("0.0.0.0/1")), t)
	psb.Subtract(sub.PrefixSet())

	if !psb.tree4.isRoot() {
		t.Fatalf("Subtract left a non-root key at the tree root: %v", psb.tree4.key)
	}
	assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), pfxs("128.0.0.0/1"))

	// The builder must remain safe to mutate after subtraction.
	tErr(psb.Remove(pfx("128.0.0.0/1")), t)
	if got := psb.PrefixSet().Size(); got != 0 {
		t.Fatalf("Size() after removing the remainder = %d, want 0", got)
	}
}

func TestPrefixSetBuilderIntersect(t *testing.T) {
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

		// Exercise case where t has only left child and o has only right child
		// (see [tree.intersectTreeImpl])
		{pfxs("::0/1"), pfxs("8000::/1"), pfxs()},

		// IPv4
		{pfxs("1.2.3.0/24"), pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.0/24"), pfxs("1.2.0.0/32"), pfxs()},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.0/24"), pfxs("::ffff:1.2.3.4/128"), pfxs()},

		// Default routes
		{pfxs("::0/0"), pfxs("::0/0"), pfxs("::0/0")},
		{pfxs("::0/0"), pfxs("::0/1"), pfxs("::0/1")},
		{pfxs("::0/0"), pfxs("::1/128"), pfxs("::1/128")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1"), pfxs("0.0.0.0/1")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.1/32"), pfxs("0.0.0.1/32")},

		// a:    0b1*, 0b110*
		// b:    0b11*
		// want: 0b11*, 0b110*
		{
			a:    pfxs("128.0.0.0/1", "192.0.0.0/3"),
			b:    pfxs("192.0.0.0/2"),
			want: pfxs("192.0.0.0/2", "192.0.0.0/3"),
		},

		// a:    0b1*, 0b110*
		// b:    0b111*
		// want: 0b111*
		// (0b110* is not encompassed by b)
		{
			a:    pfxs("128.0.0.0/1", "192.0.0.0/3"),
			b:    pfxs("224.0.0.0/3"),
			want: pfxs("224.0.0.0/3"),
		},

		// a:    0b1*, 0b110*
		// b:    0b1110*
		// want: 0b1110*
		// (0b110* is not encompassed by b)
		{
			a:    pfxs("128.0.0.0/1", "192.0.0.0/3"),
			b:    pfxs("224.0.0.0/4"),
			want: pfxs("224.0.0.0/4"),
		},

		// Examples from property-based test that failed in old implementation
		{
			a:    pfxs("224.0.0.0/3", "224.0.0.0/6"),
			b:    pfxs("234.0.0.0/28", "206.0.0.0/8"),
			want: pfxs("234.0.0.0/28"),
		},
		{
			a:    pfxs("224.0.0.0/3", "224.0.0.0/6"),
			b:    pfxs("206.0.0.0/8", "128.0.0.0/23", "234.0.0.0/28"),
			want: pfxs("234.0.0.0/28"),
		},
		{
			a:    pfxs("0.0.0.0/2", "0.0.0.0/0", "181.168.80.0/20"),
			b:    pfxs("24.192.0.0/10", "0.0.0.0/1", "177.1.2.6/32"),
			want: pfxs("0.0.0.0/1", "0.0.0.0/2", "24.192.0.0/10", "177.1.2.6/32"),
		},
	}
	performTest := func(x, y []netip.Prefix, want []netip.Prefix) {
		psb := &PrefixSetBuilder{}
		for _, p := range x {
			tErr(psb.Add(p), t)
		}
		intersectPsb := &PrefixSetBuilder{}
		for _, p := range y {
			tErr(intersectPsb.Add(p), t)
		}
		psb.Intersect(intersectPsb.PrefixSet())
		assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), want)
	}

	for _, tt := range tests {
		performTest(tt.a, tt.b, tt.want)
		performTest(tt.b, tt.a, tt.want)
	}
}

func TestPrefixSetBuilderMerge(t *testing.T) {
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
			pfxs("1.2.3.4/32", "::ffff:1.2.3.4/128"),
		},

		// Default routes
		{pfxs("::0/0"), pfxs(), pfxs("::0/0")},
		{pfxs("::0/0"), pfxs("::0/1"), pfxs("::0/0", "::0/1")},
		{pfxs("0.0.0.0/0"), pfxs(), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1"), pfxs("0.0.0.0/0", "0.0.0.0/1")},

		// Ensure children are preserved in zero-overlap scenario
		// ("neither is a prefix of the other" case).
		{
			pfxs("10.0.0.0/16"),
			pfxs("20.0.0.0/16", "20.0.1.0/24"), // parent + child
			pfxs("10.0.0.0/16", "20.0.0.0/16", "20.0.1.0/24"),
		},
	}
	performTest := func(x, y []netip.Prefix, want []netip.Prefix) {
		psb := &PrefixSetBuilder{}
		for _, p := range x {
			tErr(psb.Add(p), t)
		}
		unionPsb := &PrefixSetBuilder{}
		for _, p := range y {
			tErr(unionPsb.Add(p), t)
		}
		psb.Merge(unionPsb.PrefixSet())
		assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), want)
	}
	for _, tt := range tests {
		performTest(tt.a, tt.b, tt.want)
		performTest(tt.b, tt.a, tt.want)
	}
}

func TestPrefixSetBuilderRemove(t *testing.T) {
	tests := []struct {
		add    []netip.Prefix
		remove []netip.Prefix
		want   []netip.Prefix
	}{
		{pfxs(), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs(), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128")},

		// Remove removes exact prefix, not entire subnet
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/128")},

		// IPv4
		{pfxs("1.2.3.4/32"), pfxs(), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32"), pfxs()},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.4/32"), pfxs("::ffff:1.2.3.4/128"), pfxs("1.2.3.4/32")},

		// Remove a node that has left == nil && right != nil (not covered by
		// any other test case)
		{pfxs("8000::/1", "C000::/2"), pfxs("8000::/1"), pfxs("C000::/2")},

		// Default routes
		{pfxs("::0/0"), pfxs("::0/0"), pfxs()},
		{pfxs("::0/0"), pfxs("::0/1"), pfxs("::0/0")},
		{pfxs("::0/0", "::0/1"), pfxs("::0/1"), pfxs("::0/0")},
		{pfxs("::0/0", "::0/1"), pfxs("::0/0"), pfxs("::0/1")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0"), pfxs()},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0", "0.0.0.0/1"), pfxs("0.0.0.0/1"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0", "0.0.0.0/1"), pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			tErr(psb.Add(p), t)
		}
		for _, p := range tt.remove {
			tErr(psb.Remove(p), t)
		}
		ps := psb.PrefixSet()
		assertEqualPrefixSlice(t, ps.Prefixes(), tt.want)
	}
}

func TestPrefixSetBuilderFilter(t *testing.T) {
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

		// IPv4
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.0/24"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.0/24"), pfxs("1.2.3.0/32"), pfxs()},

		// IPv4-mapped IPv6 addresses are distinct from IPv4 addresses
		{pfxs("1.2.3.4/32"), pfxs("::ffff:1.2.3.4/32"), pfxs()},

		// Example from method documentation
		{pfxs("1.2.3.4/32", "1.2.0.0/16"), pfxs("1.2.3.0/24"), pfxs("1.2.3.4/32")},

		// Default routes
		{pfxs("::0/0"), pfxs("::0/0"), pfxs("::0/0")},
		{pfxs("::0/0"), pfxs("::0/1"), pfxs()},
		{pfxs("::0/1"), pfxs("::0/0"), pfxs("::0/1")},
		{pfxs("::0/0", "::0/1"), pfxs("::0/0"), pfxs("::0/0", "::0/1")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1"), pfxs()},
		{pfxs("0.0.0.0/1"), pfxs("0.0.0.0/0"), pfxs("0.0.0.0/1")},
		{pfxs("0.0.0.0/0", "0.0.0.0/1"), pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0", "0.0.0.0/1")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			tErr(psb.Add(p), t)
		}
		filterPsb := &PrefixSetBuilder{}
		for _, p := range tt.filter {
			tErr(filterPsb.Add(p), t)
		}
		psb.Filter(filterPsb.PrefixSet())
		assertEqualPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSetPrefixesCompact(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128")},
		// Note: siblings are not merged!
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
		{pfxs("1.2.3.4/32", "::ffff:1.2.3.4/128"), pfxs("1.2.3.4/32", "::ffff:1.2.3.4/128")},

		// Default routes
		{pfxs("::0/0"), pfxs("::0/0")},
		{pfxs("::0/0", "::0/1"), pfxs("::0/0")},
		{pfxs("0.0.0.0/0"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0", "0.0.0.0/1"), pfxs("0.0.0.0/0")},
		{pfxs("0.0.0.0/0", "::0/0"), pfxs("0.0.0.0/0", "::0/0")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			tErr(psb.Add(p), t)
		}
		ps := psb.PrefixSet()
		assertEqualPrefixSlice(t, ps.PrefixesCompact(), tt.want)
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

		// Default routes
		{pfxs("::0/0"), 1},
		{pfxs("0.0.0.0/0"), 1},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			tErr(psb.Add(p), t)
		}
		ps := psb.PrefixSet()
		if got := ps.Size(); got != tt.want {
			t.Errorf("ps.Size() = %d, want %d", got, tt.want)
		}
	}
}
