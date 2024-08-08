package netipds

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

func wantMap[T comparable](val T, prefixes ...string) map[netip.Prefix]T {
	m := make(map[netip.Prefix]T, len(prefixes))
	for _, pStr := range prefixes {
		p := netip.MustParsePrefix(pStr)
		m[p] = val
	}
	return m
}

func checkMap[T comparable](t *testing.T, want, got map[netip.Prefix]T) {
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
		return
	}
	for k, v := range got {
		if wantV, ok := want[k]; !ok || v != wantV {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}
}

func TestPrefixMapGet(t *testing.T) {
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
		{pfxs("::/128", "::1/128"), pfx("::1/128"), true},
		{pfxs("::/128", "::1/128", "::2/127"), pfx("::1/128"), true},
		{pfxs("::/128", "::1/128", "::2/127", "::3/127"), pfx("::1/128"), true},
		{pfxs("::/128"), pfx("::/0"), false},

		// Make sure we can't get a prefix that has a node but no entry
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},

		// Make sure parent/child insert order doesn't matter
		{pfxs("::0/127", "::0/128"), pfx("::0/127"), true},
		{pfxs("::0/128", "::0/127"), pfx("::0/127"), true},
		{pfxs("::0/128", "::0/127", "::1/128"), pfx("::0/127"), true},

		// TODO: should we allow ::/0 to be used as a key?
		{pfxs("::/0"), pfx("::/0"), false},

		// IPv4
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		// Test PrefixMapBuilder.Get()
		if _, ok := pmb.Get(tt.get); ok != tt.want {
			t.Errorf("pmb.Get(%s) = %v, want %v", tt.get, ok, tt.want)
		}
		// Test PrefixMap.Get()
		pm := pmb.PrefixMap()
		if _, ok := pm.Get(tt.get); ok != tt.want {
			t.Errorf("pm.Get(%s) = %v, want %v", tt.get, ok, tt.want)
		}
	}
}

func TestPrefixMapContains(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128", "::1/128"), pfx("::0/128"), true},
		{pfxs("::0/128", "::1/128"), pfx("::1/128"), true},
		{pfxs("::0/128", "::1/128"), pfx("::2/128"), false},

		// Nodes without entries should not report as contained
		{pfxs("::0/128", "::1/128"), pfx("::2/127"), false},

		// IPv4
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},

		// IPv4 prefixes are appropriately wrapped
		{pfxs("1.2.3.0/24"), pfx("::/24"), false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		if got := pm.Contains(tt.get); got != tt.want {
			t.Errorf("pm.Contains(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixMapContainsAfterRemove(t *testing.T) {
	tests := []struct {
		set    []netip.Prefix
		remove []netip.Prefix
		get    netip.Prefix
		want   bool
	}{
		{pfxs("::0/128"), pfxs("::0/128"), pfx("::0/128"), false},

		// Try to remove entry-less parent
		{pfxs("::0/128", "::1/128"), pfxs("::0/127"), pfx("::0/128"), true},

		// Remove a entry's parent entry
		{pfxs("::0/127", "::0/128", "::1/128"), pfxs("::0/127"), pfx("::0/128"), true},

		// Remove child of an entry
		{pfxs("::0/127", "::0/128", "::1/128"), pfxs("::0/128"), pfx("::0/127"), true},

		// IPv4
		{pfxs("1.2.3.3/32"), pfxs("1.2.3.4/32"), pfx("1.2.3.4/32"), false},
	}

	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		for _, p := range tt.remove {
			pmb.Remove(p)
		}
		pm := pmb.PrefixMap()
		if got := pm.Contains(tt.get); got != tt.want {
			t.Errorf("pm.Contains(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixMapEncompasses(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},

		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::0/127"), false},

		{pfxs("::0/127"), pfx("::0/128"), true},
		{pfxs("::0/127"), pfx("::1/128"), true},

		{pfxs("::2/127"), pfx("::1/128"), false},
		{pfxs("::2/127"), pfx("::2/128"), true},
		{pfxs("::2/127"), pfx("::3/128"), true},

		// A Prefix is not considered encompassed if the map contains all of its
		// children but not the Prefix itself.
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},

		// IPv4
		{pfxs("10.0.0.1/32"), pfx("10.0.0.1/32"), true},
		{pfxs("10.0.0.0/32"), pfx("10.0.0.0/31"), false},

		{pfxs("10.0.0.0/31"), pfx("10.0.0.0/32"), true},
		{pfxs("10.0.0.0/31"), pfx("10.0.0.1/32"), true},

		{pfxs("10.0.0.2/31"), pfx("10.0.0.1/32"), false},
		{pfxs("10.0.0.2/31"), pfx("10.0.0.2/32"), true},
		{pfxs("10.0.0.2/31"), pfx("10.0.0.3/32"), true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		if got := pm.Encompasses(tt.get); got != tt.want {
			t.Errorf("pm.Encompasses(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixMapEncompassesStrict(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},

		{pfxs("::0/128"), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), false},

		{pfxs("::0/127"), pfx("::0/128"), true},
		{pfxs("::0/127"), pfx("::1/128"), true},

		{pfxs("::2/127"), pfx("::1/128"), false},
		{pfxs("::2/127"), pfx("::2/128"), true},
		{pfxs("::2/127"), pfx("::3/128"), true},

		{pfxs("::0/127", "::0/128"), pfx("::0/127"), false},

		// This map contains a node that strictly encompasses the query, but
		// that node does not have an entry
		{pfxs("::0/128", "::1/128"), pfx("::0/128"), false},

		// A Prefix is not considered encompassed if the map contains all of its
		// children but not the Prefix itself.
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},

		// IPv4
		{pfxs("10.0.0.1/32"), pfx("10.0.0.1/32"), false},
		{pfxs("10.0.0.0/32"), pfx("10.0.0.0/31"), false},

		{pfxs("10.0.0.0/31"), pfx("10.0.0.0/32"), true},
		{pfxs("10.0.0.0/31"), pfx("10.0.0.1/32"), true},

		{pfxs("10.0.0.2/31"), pfx("10.0.0.1/32"), false},
		{pfxs("10.0.0.2/31"), pfx("10.0.0.2/32"), true},
		{pfxs("10.0.0.2/31"), pfx("10.0.0.3/32"), true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		if got := pm.EncompassesStrict(tt.get); got != tt.want {
			t.Errorf("pm.EncompassesStrict(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixMapToMap(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		want map[netip.Prefix]bool
	}{
		{pfxs(), wantMap(true)},
		{pfxs("::0/128"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), wantMap(true, "::1/128")},
		{pfxs("::2/128"), wantMap(true, "::2/128")},
		{pfxs("::2/127"), wantMap(true, "::2/127")},
		{pfxs("::0/128", "::1/128"), wantMap(true, "::0/128", "::1/128")},

		// Parent and children are both included if they have entries
		{pfxs("::0/127", "::0/128"), wantMap(true, "::0/127", "::0/128")},
		{pfxs("::0/127", "::0/128", "::1/128"), wantMap(true, "::0/127", "::0/128", "::1/128")},

		// IPv4
		{pfxs("10.0.0.0/32"), wantMap(true, "10.0.0.0/32")},
		{pfxs("10.0.0.1/32"), wantMap(true, "10.0.0.1/32")},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		checkMap(t, tt.want, pmb.PrefixMap().ToMap())
	}
}

func TestPrefixMapRemove(t *testing.T) {
	tests := []struct {
		set    []netip.Prefix
		remove []netip.Prefix
		want   map[netip.Prefix]bool
	}{
		{pfxs("::0/128"), pfxs("::0/128"), wantMap(true)},

		// Try to remove a node with two children and no entry
		{
			set:    pfxs("::1/128", "::0/128"),
			remove: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Remove a node wth two children and an entry
		{
			set:    pfxs("::0/127", "::1/128", "::0/128"),
			remove: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Remove a node with one child and an entry
		{
			set:    pfxs("::0/126", "::0/127", "::1/128"),
			remove: pfxs("::0/126"),
			want:   wantMap(true, "::0/127", "::1/128"),
		},

		// Remove one sibling
		{
			set:    pfxs("::0/128", "::1/128"),
			remove: pfxs("::1/128"),
			want:   wantMap(true, "::0/128"),
		},

		// Try to remove an entry-less parent and one sibling
		{
			set:    pfxs("::0/128", "::1/128"),
			remove: pfxs("::0/127", "::1/128"),
			want:   wantMap(true, "::0/128"),
		},

		// Remove two siblings with a common parent entry
		{
			set:    pfxs("::0/127", "::0/128", "::1/128"),
			remove: pfxs("::0/128", "::1/128"),
			want:   wantMap(true, "::0/127"),
		},

		// Remove two siblings with a common parent entry
		{
			set:    pfxs("::0/128", "::1/128"),
			remove: pfxs("::0/128", "::1/128"),
			want:   wantMap(true),
		},

		// IPv4
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32"), wantMap(true)},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		for _, p := range tt.remove {
			pmb.Remove(p)
		}
		checkMap(t, tt.want, pmb.PrefixMap().ToMap())
	}
}

func TestPrefixMapBuilderSubtract(t *testing.T) {
	tests := []struct {
		set      []netip.Prefix
		subtract netip.Prefix
		want     map[netip.Prefix]bool
	}{
		{pfxs(), netip.Prefix{}, wantMap(true)},
		{pfxs("::0/1"), pfx("::0/1"), wantMap(true)},
		{pfxs("::0/2"), pfx("::0/2"), wantMap(true)},
		{pfxs("::0/128"), pfx("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::0/127"), wantMap(true)},
		{pfxs("::0/128"), pfx("::1/128"), wantMap(true, "::0/128")},
		{pfxs("::0/127"), pfx("::0/128"), wantMap(true, "::1/128")},
		{pfxs("::2/127"), pfx("::3/128"), wantMap(true, "::2/128")},
		{pfxs("::0/126"), pfx("::0/128"), wantMap(true, "::1/128", "::2/127")},
		{pfxs("::0/126"), pfx("::3/128"), wantMap(true, "::0/127", "::2/128")},
		// IPv4
		{
			set:      pfxs("1.2.3.0/30"),
			subtract: pfx("1.2.3.0/32"),
			want:     wantMap(true, "1.2.3.1/32", "1.2.3.2/31"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pmb.Subtract(tt.subtract)
		checkMap(t, tt.want, pmb.PrefixMap().ToMap())
	}
}

// Make sure Subtract does not overwrite child entries as it creates nodes to
// fill in gaps.
func TestPrefixMapBuilderSubtractNoOverwrite(t *testing.T) {
	pmb := &PrefixMapBuilder[string]{}
	pmb.Set(pfx("::0/127"), "parent")
	pmb.Set(pfx("::1/128"), "child")
	pmb.Subtract(pfx("::0/128"))
	pm := pmb.PrefixMap()
	want := "child"
	if val, _ := pm.Get(pfx("::1/128")); val != want {
		t.Errorf("pm.Get(::1/128) = %v, want %v", val, want)
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

		// Unlike RootOfStrict, RootOf will return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), pfx("::0/128"), true},

		// Make sure entry-less nodes are not returned by RootOf
		{pfxs("::0/127", "::2/127"), pfx("::0/128"), pfx("::0/127"), true},

		// IPv4
		{pfxs(), pfx("1.2.3.0/32"), netip.Prefix{}, false},
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},
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

func TestPrefixMapRootOfStrict(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},

		// Unlike RootOf, RootOfStrict will not return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), netip.Prefix{}, false},

		// Make sure entry-less nodes are not returned by RootOfStrict
		{pfxs("::0/127", "::2/127"), pfx("::0/128"), pfx("::0/127"), true},

		// IPv4
		{pfxs(), pfx("1.2.3.0/32"), netip.Prefix{}, false},
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		gotPrefix, _, gotOK := pm.RootOfStrict(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"pm.RootOfStrict(%s) = (%v, _, %v), want (%v, _, %v)",
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

		// Unlike ParentOfStrict, ParentOf will return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), pfx("::0/128"), true},

		// IPv4
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), pfx("1.2.3.0/32"), true},
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

func TestPrefixMapParentOfStrict(t *testing.T) {
	tests := []struct {
		set        []netip.Prefix
		get        netip.Prefix
		wantPrefix netip.Prefix
		wantOK     bool
	}{
		{pfxs(), pfx("::0/128"), netip.Prefix{}, false},
		{pfxs("::0/127"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/127", "::0/128"), pfx("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/1"), pfx("::0/128"), pfx("::0/1"), true},

		// Unlike ParentOf, ParentOfStrict will not return the prefix itself
		{pfxs("::0/128"), pfx("::0/128"), netip.Prefix{}, false},

		// IPv4
		{pfxs("1.2.3.0/31"), pfx("1.2.3.0/32"), pfx("1.2.3.0/31"), true},
		{pfxs("128.0.0.0/1"), pfx("128.0.0.0/32"), pfx("128.0.0.0/1"), true},

		// Another strictness check
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), netip.Prefix{}, false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		gotPrefix, _, gotOK := pm.ParentOfStrict(tt.get)
		if gotPrefix != tt.wantPrefix || gotOK != tt.wantOK {
			t.Errorf(
				"pm.ParentOfStrict(%s) = (%v, _, %v), want (%v, _, %v)",
				tt.get, gotPrefix, gotOK, tt.wantPrefix, tt.wantOK,
			)
		}
	}
}

func TestPrefixMapDescendantsOf(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want map[netip.Prefix]bool
	}{
		{pfxs(), pfx("::0/128"), wantMap(true)},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), wantMap(true)},
		{pfxs("::1/128"), pfx("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::0/128"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), wantMap(true, "::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), wantMap(true, "::2/128")},
		{pfxs("::0/128"), pfx("::1/127"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), pfx("::0/127"), wantMap(true, "::1/128")},
		{pfxs("::2/127"), pfx("::2/127"), wantMap(true, "::2/127")},

		// Using "::/0" as a lookup key
		{pfxs("::0/128"), pfx("::/0"), wantMap(true, "::0/128")},

		// Get a prefix that has no entry but has children.
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: wantMap(true, "::0/128", "::1/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128", "::2/128"),
			get:  pfx("::2/127"),
			want: wantMap(true, "::2/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: wantMap(true, "::0/128", "::1/128"),
		},
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::2/127"),
			want: wantMap(true, "::2/128", "::3/128"),
		},

		// Get an entry-less shared prefix node that has an entry-less child
		{
			set: pfxs("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// entry-less shared prefixes.
			get:  pfx("::4/126"),
			want: wantMap(true, "::4/128", "::6/128", "::7/128"),
		},

		// Get a node that is both an entry and a shared prefix node and has an
		// entry-less child
		{
			set: pfxs("::4/126", "::6/128", "::7/128"),
			get: pfx("::4/126"),
			// The node "::6/127" is a node in the tree but has no entry, so it
			// should not be included in the result.
			want: wantMap(true, "::4/126", "::6/128", "::7/128"),
		},

		// Get a prefix that has no exact node, but still has descendants
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::0/126"),
			want: wantMap(true, "::2/128", "::3/128"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), wantMap(true, "1.2.3.0/32")},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/24"), wantMap(true, "1.2.3.0/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.3.0/24"), wantMap(true, "1.2.3.1/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.4.0/24"), wantMap(true)},
		{
			set:  pfxs("1.2.3.0/32", "1.2.3.1/32"),
			get:  pfx("1.2.3.0/24"),
			want: wantMap(true, "1.2.3.0/32", "1.2.3.1/32"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		checkMap(t, tt.want, pmb.PrefixMap().DescendantsOf(tt.get).ToMap())
	}
}

func TestPrefixMapDescendantsOfStrict(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want map[netip.Prefix]bool
	}{
		{pfxs(), pfx("::0/128"), wantMap(true)},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), wantMap(true)},
		{pfxs("::1/128"), pfx("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::0/128"), wantMap(true)},
		{pfxs("::1/128"), pfx("::1/128"), wantMap(true)},
		{pfxs("::2/128"), pfx("::2/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::1/127"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), pfx("::0/127"), wantMap(true, "::1/128")},
		{pfxs("::2/127"), pfx("::2/127"), wantMap(true)},

		// Multi-prefix map
		{
			set:  pfxs("::0/127", "::0/128"),
			get:  pfx("::0/127"),
			want: wantMap(true, "::0/128"),
		},

		// Using "::/0" as a lookup key
		{pfxs("::0/128"), pfx("::/0"), wantMap(true, "::0/128")},

		// Get a prefix that has no entry but has children.
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: wantMap(true, "::0/128", "::1/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128", "::2/128"),
			get:  pfx("::2/127"),
			want: wantMap(true, "::2/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/127"),
			want: wantMap(true, "::0/128", "::1/128"),
		},
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::2/127"),
			want: wantMap(true, "::2/128", "::3/128"),
		},

		// Get a entry-less shared prefix node that has a entry-less child
		{
			set: pfxs("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// entry-less shared prefixes.
			get:  pfx("::4/126"),
			want: wantMap(true, "::4/128", "::6/128", "::7/128"),
		},

		// Get an entry shared prefix node that has a entry-less child
		{
			set: pfxs("::4/126", "::6/128", "::7/128"),
			get: pfx("::4/126"),
			// The node "::6/127" is a node in the tree but has no entry, so it
			// should not be included in the result.
			want: wantMap(true, "::6/128", "::7/128"),
		},

		// Get a prefix that has no exact node, but still has descendants
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::0/126"),
			want: wantMap(true, "::2/128", "::3/128"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), wantMap(true)},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/24"), wantMap(true, "1.2.3.0/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.3.0/24"), wantMap(true, "1.2.3.1/32")},
		{pfxs("1.2.3.1/32"), pfx("1.2.4.0/24"), wantMap(true)},
		{
			set:  pfxs("1.2.3.0/32", "1.2.3.1/32"),
			get:  pfx("1.2.3.0/24"),
			want: wantMap(true, "1.2.3.0/32", "1.2.3.1/32"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		checkMap(t, tt.want, pmb.PrefixMap().DescendantsOfStrict(tt.get).ToMap())
	}
}

func TestPrefixMapAncestorsOf(t *testing.T) {
	result := func(prefixes ...string) map[netip.Prefix]bool {
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
		{pfxs(), pfx("::0/128"), result()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), result()},
		{pfxs("::1/128"), pfx("::0/128"), result()},
		{pfxs("::0/128"), pfx("::0/128"), result("::0/128")},
		{pfxs("::1/128"), pfx("::1/128"), result("::1/128")},
		{pfxs("::2/128"), pfx("::2/128"), result("::2/128")},
		{pfxs("::0/127"), pfx("::0/128"), result("::0/127")},
		{pfxs("::0/127"), pfx("::1/128"), result("::0/127")},
		{pfxs("::2/127"), pfx("::2/127"), result("::2/127")},

		// Multi-prefix maps
		{
			set:  pfxs("::0/127", "::0/128"),
			get:  pfx("::0/128"),
			want: result("::0/127", "::0/128"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/128"),
			want: result("::0/128"),
		},
		{
			set:  pfxs("::0/126", "::0/127", "::1/128"),
			get:  pfx("::0/128"),
			want: result("::0/126", "::0/127"),
		},

		// Make sure nodes without entries are excluded
		{
			set: pfxs("::0/128", "::2/128"),
			get: pfx("::0/128"),
			// "::2/127" is a node in the tree but has no entry, so it should
			// not be included in the result.
			want: result("::0/128"),
		},

		// Make sure parent/child insertion order doesn't matter
		{
			set:  pfxs("::0/126", "::0/127"),
			get:  pfx("::0/128"),
			want: result("::0/127", "::0/126"),
		},
		{
			set:  pfxs("::0/127", "::0/126"),
			get:  pfx("::0/128"),
			want: result("::0/127", "::0/126"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.1/32"), result()},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), result("1.2.3.0/32")},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/32"), result("1.2.3.0/24")},
		// Insert shortest prefix first
		{
			set:  pfxs("1.2.0.0/16", "1.2.3.0/24"),
			get:  pfx("1.2.3.0/32"),
			want: result("1.2.3.0/24", "1.2.0.0/16"),
		},
		// Insert longest prefix first
		{
			set:  pfxs("1.2.3.0/24", "1.2.0.0/16"),
			get:  pfx("1.2.3.0/32"),
			want: result("1.2.3.0/24", "1.2.0.0/16"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		checkMap(t, tt.want, pmb.PrefixMap().AncestorsOf(tt.get).ToMap())
	}

}

func TestPrefixMapAncestorsOfStrict(t *testing.T) {
	result := func(prefixes ...string) map[netip.Prefix]bool {
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
		{pfxs(), pfx("::0/128"), result()},

		// Single-prefix maps
		{pfxs("::0/128"), pfx("::1/128"), result()},
		{pfxs("::1/128"), pfx("::0/128"), result()},
		{pfxs("::0/128"), pfx("::0/128"), result()},
		{pfxs("::1/128"), pfx("::1/128"), result()},
		{pfxs("::2/128"), pfx("::2/128"), result()},
		{pfxs("::0/127"), pfx("::0/128"), result("::0/127")},
		{pfxs("::0/127"), pfx("::1/128"), result("::0/127")},
		{pfxs("::2/127"), pfx("::2/127"), result()},

		// Multi-prefix maps
		{
			set:  pfxs("::0/127", "::0/128"),
			get:  pfx("::0/128"),
			want: result("::0/127"),
		},
		{
			set:  pfxs("::0/128", "::1/128"),
			get:  pfx("::0/128"),
			want: result(),
		},
		{
			set:  pfxs("::0/126", "::0/127", "::1/128"),
			get:  pfx("::0/128"),
			want: result("::0/126", "::0/127"),
		},

		// Make sure nodes without entries are excluded
		{
			set: pfxs("::0/128", "::2/128"),
			get: pfx("::0/128"),
			// "::2/127" is a node in the tree but has no entry, so it should
			// not be included in the result. "0::/128" is the prefix itself,
			// so it is also excluded.
			want: result(),
		},

		// Make sure parent/child insertion order doesn't matter
		{
			set:  pfxs("::0/126", "::0/127"),
			get:  pfx("::0/128"),
			want: result("::0/127", "::0/126"),
		},
		{
			set:  pfxs("::0/127", "::0/126"),
			get:  pfx("::0/128"),
			want: result("::0/127", "::0/126"),
		},

		// IPv4
		{pfxs("1.2.3.0/32"), pfx("1.2.3.1/32"), result()},
		{pfxs("1.2.3.0/32"), pfx("1.2.3.0/32"), result()},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/32"), result("1.2.3.0/24")},
		// Insert shortest prefix first
		{
			set:  pfxs("1.2.0.0/16", "1.2.3.0/24"),
			get:  pfx("1.2.3.0/32"),
			want: result("1.2.3.0/24", "1.2.0.0/16"),
		},
		// Insert longest prefix first
		{
			set:  pfxs("1.2.3.0/24", "1.2.0.0/16"),
			get:  pfx("1.2.3.0/32"),
			want: result("1.2.3.0/24", "1.2.0.0/16"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		checkMap(t, tt.want, pmb.PrefixMap().AncestorsOfStrict(tt.get).ToMap())
	}

}
func TestPrefixMapBuilderUsableAfterPrefixMap(t *testing.T) {
	pmb := &PrefixMapBuilder[int]{}

	// Create initial map
	pmb.Set(pfx("::0/128"), 1)
	pmb.Set(pfx("::1/128"), 1)
	pm1 := pmb.PrefixMap()

	// Make modifications with the sam PrefixMapBuilder and create a new map
	pmb.Remove(pfx("::0/128"))
	pmb.Set(pfx("::1/128"), 2)
	pmb.Set(pfx("::2/128"), 2)
	pm2 := pmb.PrefixMap()

	checkMap(t, wantMap(1, "::0/128", "::1/128"), pm1.ToMap())
	checkMap(t, wantMap(2, "::1/128", "::2/128"), pm2.ToMap())
}

func TestPrefixMapBuilderFilter(t *testing.T) {
	tests := []struct {
		set    []netip.Prefix
		filter []netip.Prefix
		want   map[netip.Prefix]bool
	}{
		{pfxs(), pfxs(), wantMap(true)},
		{pfxs(), pfxs("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfxs(), wantMap(true)},

		{pfxs("::0/128"), pfxs("::0/128"), wantMap(true, "::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), pfxs("::0/127"), wantMap(true, "::1/128")},

		// Filter by one of the entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/128"),
			want:   wantMap(true, "::0/128"),
		},

		// Filter by a parent of all entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filter by a parent of some entries in the map
		{
			set:    pfxs("::0/128", "::1/128", "::2/128"),
			filter: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filter by all entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/128", "::1/128"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filtering uses encompassment; the filter covers "::0/127" but does
		// not encompass it.
		{pfxs("::0/127"), pfxs("::0/128", "::1/128"), wantMap(true)},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		filter := &PrefixSetBuilder{}
		for _, p := range tt.filter {
			filter.Add(p)
		}
		pmb.Filter(filter.PrefixSet())
		checkMap(t, tt.want, pmb.PrefixMap().ToMap())
	}
}

func TestPrefixMapFilter(t *testing.T) {
	tests := []struct {
		set    []netip.Prefix
		filter []netip.Prefix
		want   map[netip.Prefix]bool
	}{
		{pfxs(), pfxs(), wantMap(true)},
		{pfxs(), pfxs("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfxs(), wantMap(true)},

		{pfxs("::0/128"), pfxs("::0/128"), wantMap(true, "::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), wantMap(true, "::0/128")},
		{pfxs("::1/128"), pfxs("::0/127"), wantMap(true, "::1/128")},

		// Filter by one of the entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/128"),
			want:   wantMap(true, "::0/128"),
		},

		// Filter by a parent of all entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filter by a parent of some entries in the map
		{
			set:    pfxs("::0/128", "::1/128", "::2/128"),
			filter: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filter by all entries in the map
		{
			set:    pfxs("::0/128", "::1/128"),
			filter: pfxs("::0/128", "::1/128"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Filtering uses encompassment; the filter covers "::0/127" but does
		// not encompass it.
		{pfxs("::0/127"), pfxs("::0/128", "::1/128"), wantMap(true)},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		filter := &PrefixSetBuilder{}
		for _, p := range tt.filter {
			filter.Add(p)
		}
		pm := pmb.PrefixMap()
		filtered := pm.Filter(filter.PrefixSet())
		checkMap(t, tt.want, filtered.ToMap())
	}
}

func TestOverlapsPrefix(t *testing.T) {
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

		// Make sure entry-less nodes don't count. This map contains
		// the shared prefix ::0/126.
		{pfxs("::0/128", "::2/128"), pfx("::3/128"), false},

		// IPv4
		{pfxs(), pfx("1.2.3.0/32"), false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		if got := pm.OverlapsPrefix(tt.get); got != tt.want {
			t.Errorf("pm.OverlapsPrefix(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixMapSize(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want int
	}{
		{pfxs(), 0},
		{pfxs("::0/128"), 1},
		{pfxs("::0/128", "::0/128"), 1},
		{pfxs("::0/128", "::1/128"), 2},
		{pfxs("::0/127", "::0/128"), 2},
		{pfxs("::0/126", "::0/127"), 2},
		{pfxs("0::0/127", "::0/128", "::1/128"), 3},
	}
	for _, tt := range tests {
		psb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.add {
			psb.Set(p, true)
		}
		ps := psb.PrefixMap()
		if got := ps.Size(); got != tt.want {
			t.Errorf("pm.Size() = %d, want %d", got, tt.want)
		}
	}
}
