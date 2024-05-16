package netipmap

import (
	"net/netip"
	"testing"
)

func prefixes(strings ...string) []netip.Prefix {
	ps := make([]netip.Prefix, len(strings))
	for i, s := range strings {
		ps[i] = netip.MustParsePrefix(s)
	}
	return ps
}

func TestPrefixMapSetGet(t *testing.T) {
	tests := []struct {
		setPrefixes []netip.Prefix
		getPrefix   string
		want        bool
	}{
		{prefixes(), "::0/128", false},
		{prefixes("::0/128"), "::0/128", true},
		{prefixes("::1/128"), "::1/128", true},
		{prefixes("::2/128"), "::2/128", true},
		{prefixes("::2/127"), "::2/127", true},
		{prefixes("1.2.3.0/24"), "1.2.3.0/24", true},
		{prefixes("1.2.3.0/24"), "1.2.3.4/32", false},
		{prefixes("::/128", "::1/128", "::2/127", "::3/127"), "::1/128", true},
		{prefixes("::/128"), "::/0", false},

		// TODO: should we allow ::/0 to be used as a key?
		{prefixes("::/0"), "::/0", false},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.setPrefixes {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		p := netip.MustParsePrefix(tt.getPrefix)
		if _, ok := pm.Get(p); ok != tt.want {
			t.Errorf("pm.Get(%s) = %v, want %v", p, ok, tt.want)
		}
	}
}

func TestPrefixMapGetDescendants(t *testing.T) {
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
		get  string
		want map[netip.Prefix]bool
	}{
		{prefixes(), "::0/128", resultMap()},

		// Single-prefix maps
		{prefixes("::0/128"), "::1/128", resultMap()},
		{prefixes("::1/128"), "::0/128", resultMap()},
		{prefixes("::0/128"), "::0/128", resultMap("::0/128")},
		{prefixes("::1/128"), "::1/128", resultMap("::1/128")},
		{prefixes("::2/128"), "::2/128", resultMap("::2/128")},
		{prefixes("::0/128"), "::1/127", resultMap("::0/128")},
		{prefixes("::1/128"), "::0/127", resultMap("::1/128")},
		{prefixes("::2/127"), "::2/127", resultMap("::2/127")},

		// Using "::/0" as a lookup key
		{prefixes("::0/128"), "::/0", resultMap("::0/128")},

		// Get a prefix that has no value but has children.
		{
			set:  prefixes("::0/128", "::1/128"),
			get:  "::0/127",
			want: resultMap("::0/128", "::1/128"),
		},
		{
			set:  prefixes("::0/128", "::1/128", "::2/128"),
			get:  "::2/127",
			want: resultMap("::2/128"),
		},
		{
			set:  prefixes("::0/128", "::1/128"),
			get:  "::0/127",
			want: resultMap("::0/128", "::1/128"),
		},
		{
			set:  prefixes("::2/128", "::3/128"),
			get:  "::2/127",
			want: resultMap("::2/128", "::3/128"),
		},

		// Get a value-less shared prefix that has a value-less child
		{
			set: prefixes("::4/128", "::6/128", "::7/128"),
			// This node is in the tree, as is "::6/127", but they are both
			// value-less shared prefixes.
			get:  "::4/126",
			want: resultMap("::4/128", "::6/128", "::7/128"),
		},

		// Get a value-ful shared prefix that has a value-less child
		{
			set: prefixes("::4/126", "::6/128", "::7/128"),
			get: "::4/126",
			// The node "::6/127" is in the tree but has no value, so it
			// should not be included in the result.
			want: resultMap("::4/126", "::6/128", "::7/128"),
		},

		// Get a shared prefix that also has a value
		{
			set:  prefixes("::2/127", "::2/128", "::3/128"),
			get:  "::2/127",
			want: resultMap("::2/127", "::2/128", "::3/128"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, p := range tt.set {
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()

		pm.root.prettyPrint("", "")
		p := netip.MustParsePrefix(tt.get)
		got := pm.GetDescendants(p)
		if len(got) != len(tt.want) {
			t.Errorf("pm.GetDescendants(%s) = %v, want %v", p, got, tt.want)
			continue
		}
		for k, v := range got {
			if wantV, ok := tt.want[k]; !ok || v != wantV {
				t.Errorf("pm.GetDescendants(%s) = %v, want %v", p, got, tt.want)
				break
			}
		}
	}
}
