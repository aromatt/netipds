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
		t.Errorf("pm.ToMap() = %v, want %v", got, want)
		return
	}
	for k, v := range got {
		if wantV, ok := want[k]; !ok || v != wantV {
			t.Errorf("pm.ToMap() = %v, want %v", got, want)
			return
		}
	}
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

		// Nodes with no values should not report as contained
		{pfxs("::0/128", "::1/128"), pfx("::2/127"), false},
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

		// Try to remove value-less parent
		{pfxs("::0/128", "::1/128"), pfxs("::0/127"), pfx("::0/128"), true},

		// Remove value-ful parent
		{pfxs("::0/127", "::0/128", "::1/128"), pfxs("::0/127"), pfx("::0/128"), true},

		// Remove child of value-fal parent
		{pfxs("::0/127", "::0/128", "::1/128"), pfxs("::0/128"), pfx("::0/127"), true},
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

		// Parent and children are both included if they have values
		{pfxs("::0/127", "::0/128"), wantMap(true, "::0/127", "::0/128")},
		{pfxs("::0/127", "::0/128", "::1/128"), wantMap(true, "::0/127", "::0/128", "::1/128")},
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

		// Try to remove a node with two children and no value
		{
			set:    pfxs("::1/128", "::0/128"),
			remove: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Remove a node wth two children and a value
		{
			set:    pfxs("::0/127", "::1/128", "::0/128"),
			remove: pfxs("::0/127"),
			want:   wantMap(true, "::0/128", "::1/128"),
		},

		// Remove a node with one child and a value
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

		// Try to remove a value-less parent and one sibling
		{
			set:    pfxs("::0/128", "::1/128"),
			remove: pfxs("::0/127", "::1/128"),
			want:   wantMap(true, "::0/128"),
		},

		// Remove two siblings with a common value-ful parent
		{
			set:    pfxs("::0/127", "::0/128", "::1/128"),
			remove: pfxs("::0/128", "::1/128"),
			want:   wantMap(true, "::0/127"),
		},

		// Remove two siblings with a common value-less parent
		{
			set:    pfxs("::0/128", "::1/128"),
			remove: pfxs("::0/128", "::1/128"),
			want:   wantMap(true),
		},
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
func TestPrefixMapDescendantsOf(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want map[netip.Prefix]bool
	}{
		//{pfxs(), pfx("::0/128"), wantMap(true)},

		//// Single-prefix maps
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

		// Get a prefix that has no value but has children.
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

		// Get a value-less shared prefix node that has a value-less child
		{
			set: pfxs("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// value-less shared prefixes.
			get:  pfx("::4/126"),
			want: wantMap(true, "::4/128", "::6/128", "::7/128"),
		},

		// Get a value-ful shared prefix node that has a value-less child
		{
			set: pfxs("::4/126", "::6/128", "::7/128"),
			get: pfx("::4/126"),
			// The node "::6/127" is a node in the tree but has no value, so it
			// should not be included in the result.
			want: wantMap(true, "::4/126", "::6/128", "::7/128"),
		},

		// Get a prefix that has no exact node, but still has descendants
		{
			set:  pfxs("::2/128", "::3/128"),
			get:  pfx("::0/126"),
			want: wantMap(true, "::2/128", "::3/128"),
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
		checkMap(t, tt.want, pmb.PrefixMap().AncestorsOf(tt.get).ToMap())
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
		sPmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			sPmb.Set(p, true)
		}
		fPmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.filter {
			fPmb.Set(p, true)
		}
		sPmb.Filter(fPmb.PrefixMap())
		checkMap(t, tt.want, sPmb.PrefixMap().ToMap())
	}
}

func TestPrefixMapRemoveDescendants(t *testing.T) {
	tests := []struct {
		set    []netip.Prefix
		remove netip.Prefix
		want   map[netip.Prefix]bool
	}{
		{pfxs(), pfx("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::0/128"), wantMap(true)},
		{pfxs("::0/128"), pfx("::1/128"), wantMap(true, "::0/128")},
		{pfxs("::0/128"), pfx("::0/127"), wantMap(true)},
		{pfxs("::0/127"), pfx("::0/128"), wantMap(true, "::1/128")},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pmb.RemoveDescendants(tt.remove)
		checkMap(t, tt.want, pmb.PrefixMap().ToMap())
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
