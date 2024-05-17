package netipmap

import (
	"net/netip"
	"testing"
)

func pfx(s string) netip.Prefix {
	return netip.MustParsePrefix(s)
}

func pfxs(strings ...string) []netip.Prefix {
	ps := make([]netip.Prefix, len(strings))
	for i, s := range strings {
		ps[i] = netip.MustParsePrefix(s)
	}
	return ps
}

func TestPrefixMapSetGet(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::1/128"), pfx("::1/128"), true},
		{pfxs("::2/128"), pfx("::2/128"), true},
		{pfxs("::2/127"), pfx("::2/127"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},
		{pfxs("::/128", "::1/128", "::2/127", "::3/127"), pfx("::1/128"), true},
		{pfxs("::/128"), pfx("::/0"), false},

		// Make sure we can't get a prefix that has a node but no value
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},

		// TODO: should we allow ::/0 to be used as a key?
		{pfxs("::/0"), pfx("::/0"), false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		if _, ok := pm.Get(tt.get); ok != tt.want {
			t.Errorf("pm.Get(%s) = %v, want %v", tt.get, ok, tt.want)
		}
	}
}

func TestPrefixMapDescendantsOf(t *testing.T) {
	resultMap := func(prefixes ...string) map[netip.Prefix]bool {
		m := make(map[netip.Prefix]bool, len(prefixes))
		for _, pStr := range prefixes {
			p := netip.MustParsePrefix(pStr)
			m[p] = true
		}
		return m
	}

	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want map[netip.Prefix]bool
	}{
		{pfxs(), pfx("::0/128"), resultMap()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), resultMap()},
		{pfxs("::1/128"), pfx("::0/128"), resultMap()},
		{pfxs("::0/128"), pfx("::0/128"), resultMap("::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), resultMap("::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), resultMap("::2/128")},
		{pfxs("::0/128"), pfx("::1/127"), resultMap("::0/128")},
		{pfxs("::1/128"), pfx("::0/127"), resultMap("::1/128")},
		{pfxs("::2/127"), pfx("::2/127"), resultMap("::2/127")},

		// Using "::/0" as a lookup key
		{pfxs("::0/128"), pfx("::/0"), resultMap("::0/128")},

		// Get a prefix that has no value but has children.
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: resultMap("::0/128", "::1/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128", "::2/128"),
			get:  pfx("::2/127"),
			want: resultMap("::2/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: resultMap("::0/128", "::1/128"),
		},
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::2/127"),
			want: resultMap("::2/128", "::3/128"),
		},

		// Get a value-less shared prefix node that has a value-less child
		{
			set: pfxs("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// value-less shared prefixes.
			get:  pfx("::4/126"),
			want: resultMap("::4/128", "::6/128", "::7/128"),
		},

		// Get a value-ful shared prefix node that has a value-less child
		{
			set: pfxs("::4/126", "::6/128", "::7/128"),
			get: pfx("::4/126"),
			// The node "::6/127" is a node in the tree but has no value, so it
			// should not be included in the result.
			want: resultMap("::4/126", "::6/128", "::7/128"),
		},

		// Get a prefix that has no exact node, but still has descendants
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::0/126"),
			want: resultMap("::2/128", "::3/128"),
		},

		// Get
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()

		pm.tree.prettyPrint("", "")
		got := pm.DescendantsOf(tt.get)
		if len(got) != len(tt.want) {
			t.Errorf("pm.GetDescendants(%s) = %v, want %v", tt.get, got, tt.want)
			continue
		}
		for k, v := range got {
			if wantV, ok := tt.want[k]; !ok || v != wantV {
				t.Errorf("pm.GetDescendants(%s) = %v, want %v", tt.get, got, tt.want)
				break
			}
		}
	}
}

func TestPrefixMapAncestorsOf(t *testing.T) {
	resultMap := func(prefixes ...string) map[netip.Prefix]bool {
		m := make(map[netip.Prefix]bool, len(prefixes))
		for _, pStr := range prefixes {
			p := netip.MustParsePrefix(pStr)
			m[p] = true
		}
		return m
	}

	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want map[netip.Prefix]bool
	}{
		{pfxs(), pfx("::0/128"), resultMap()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), resultMap()},
		{pfxs("::1/128"), pfx("::0/128"), resultMap()},
		{pfxs("::0/128"), pfx("::0/128"), resultMap("::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), resultMap("::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), resultMap("::2/128")},
		{pfxs("::0/127"), pfx("::0/128"), resultMap("::0/127")},
		{pfxs("::0/127"), pfx("::1/128"), resultMap("::0/127")},
		{pfxs("::2/127"), pfx("::2/127"), resultMap("::2/127")},

		{
			set:  pfxs("::0/127", "::0/128"),
			get:  pfx("::0/128"),
			want: resultMap("::0/127", "::0/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/128"),
			want: resultMap("::0/128"),
		},
		{
			set:  pfxs("::0/126", "::0/127", "::1/128"),
			get:  pfx("::0/128"),
			want: resultMap("::0/126", "::0/127"),
		},

		// Make sure nodes with no values are excluded
		{
			set: pfxs("::0/128", "::2/128"),
			get: pfx("::0/128"),
			// "::2/127" is a node in the tree but has no value, so it should
			// not be included in the result.
			want: resultMap("::0/128"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		got := pm.AncestorsOf(tt.get)
		if len(got) != len(tt.want) {
			t.Errorf("pm.GetAncestors(%s) = %v, want %v", tt.get, got, tt.want)
			continue
		}
		for k, v := range got {
			if wantV, ok := tt.want[k]; !ok || v != wantV {
				t.Errorf("pm.GetAncestors(%s) = %v, want %v", tt.get, got, tt.want)
				break
			}
		}
	}
}

func TestPrefixMapRootOf(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},

		// Make sure value-less nodes are not returned by rootOf
		{pfxs("::0/127", "::2/127"), pfx("::0/128"), pfx("::0/127"), true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		gotPrefix, _, gotOK := pm.RootOf(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"pm.RootOf(%s) = (%v, _, %v), want (%v, _, %v)",
				tt.get, gotPrefix, gotOK, tt.wantPrefix, tt.wantOK,
			)
		}
	}
}

func TestPrefixMapParentOf(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},
		{pfxs("::0/128"), pfx("::0/128"), pfx("::0/128"), true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		gotPrefix, _, gotOK := pm.ParentOf(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"pm.ParentOf(%s) = (%v, _, %v), want (%v, _, %v)",
				tt.get, gotPrefix, gotOK, tt.wantPrefix, tt.wantOK,
			)
		}
	}
}
