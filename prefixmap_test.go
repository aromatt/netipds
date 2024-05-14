package netipmap

import (
	"net/netip"
	"testing"
)

func TestGetBit(t *testing.T) {
	tests := []struct {
		a    uint128
		i    uint8
		want bool
	}{
		{uint128{0, 0}, 0, false},
		{uint128{0, 1}, 0, false},
		{uint128{1 << 63, 0}, 0, true},
		{uint128{1 << 62, 0}, 1, true},
		{uint128{0, 1 << 63}, 64, true},
		{uint128{0, 1}, 127, true},
		{uint128{0, 2}, 126, true},
		{uint128{^uint64(0), ^uint64(0)}, 0, true},
		{uint128{^uint64(0), ^uint64(0)}, 127, true},
		{uint128{^uint64(0), ^uint64(0)}, 128, false},
	}
	for _, tt := range tests {
		if got := getBit(tt.a, tt.i); got != tt.want {
			t.Errorf("getBit(%v, %d) = %v, want %v", tt.a, tt.i, got, tt.want)
		}
	}
}

func TestBasicPrefixMap(t *testing.T) {
	tests := []struct {
		setPrefixes []string
		getPrefixes []string
	}{
		{[]string{"0::0/127", "0::1/128"}, []string{"0::1/128"}},
	}
	for _, tt := range tests {
		pmb := &PrefixMapBuilder[bool]{}
		for _, pStr := range tt.setPrefixes {
			p := netip.MustParsePrefix(pStr)
			pmb.Set(p, true)
		}
		pm := pmb.PrefixMap()
		for _, pStr := range tt.getPrefixes {
			p := netip.MustParsePrefix(pStr)
			if _, ok := pm.Get(p); !ok {
				t.Errorf("pm.Get(%v) = (_, false); want (_, true)", p)
			}
		}
	}
}
