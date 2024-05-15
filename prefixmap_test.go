package netipmap

import (
	"net/netip"
	"testing"
)

func TestPrefixMapSetGet(t *testing.T) {
	tests := []struct {
		setPrefixes []string
		getPrefix   string
		want        bool
	}{
		{[]string{}, "0::0/128", false},
		{[]string{"0::0/128"}, "0::0/128", true},
		{[]string{"0::1/128"}, "0::1/128", true},
		{[]string{"0::2/128"}, "0::2/128", true},
		{[]string{"0::2/127"}, "0::2/127", true},
		{[]string{"1.2.3.0/24"}, "1.2.3.0/24", true},
		{[]string{"1.2.3.0/24"}, "1.2.3.4/32", false},
		{[]string{"0::0/128", "0::1/128", "0::2/127", "0::3/127"}, "0::1/128", true},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, pStr := range tt.setPrefixes {
			p := netip.MustParsePrefix(pStr)
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		p := netip.MustParsePrefix(tt.getPrefix)
		if _, ok := pm.Get(p); ok != tt.want {
			t.Errorf("pm.Get(%s) = %v, want %v", p, ok, tt.want)
		}
	}
}
