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

func TestIsPrefixOf(t *testing.T) {
	tests := []struct {
		a    label
		b    label
		want bool
	}{
		{NewLabel(uint128{0, 0}, 0), NewLabel(uint128{0, 0}, 0), true},
		{NewLabel(uint128{0, 0}, 0), NewLabel(uint128{0, 0}, 1), true},
		{NewLabel(uint128{0, 2}, 127), NewLabel(uint128{0, 3}, 128), true},
		{NewLabel(uint128{1, 2}, 127), NewLabel(uint128{1, 3}, 128), true},
		{NewLabel(uint128{1, 0}, 64), NewLabel(uint128{1, 1}, 128), true},
		{NewLabel(uint128{1 << 63, 0}, 1), NewLabel(uint128{1 << 63, 1}, 128), true},
		{NewLabel(uint128{1 << 63, 0}, 1), NewLabel(uint128{0, 1}, 128), false},
	}
	for _, tt := range tests {
		if got := tt.a.isPrefixOf(tt.b); got != tt.want {
			t.Errorf("%v.isPrefixOf(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPrefixUnion(t *testing.T) {
	tests := []struct {
		a    label
		b    label
		want uint8
	}{
		{NewLabel(uint128{0, 0}, 0), NewLabel(uint128{0, 0}, 0), 0},
		{NewLabel(uint128{0, 0}, 0), NewLabel(uint128{0, 0}, 1), 1},
		{NewLabel(uint128{0, 0}, 1), NewLabel(uint128{0, 0}, 0), 1},
		{NewLabel(uint128{0, 0}, 1), NewLabel(uint128{0, 0}, 1), 1},
		{NewLabel(uint128{0, 0}, 1), NewLabel(uint128{0, 0}, 2), 2},
		{NewLabel(uint128{0, 0}, 2), NewLabel(uint128{0, 0}, 1), 2},
		{NewLabel(uint128{0, 0}, 2), NewLabel(uint128{0, 0}, 2), 2},
		{NewLabel(uint128{0, 0}, 127), NewLabel(uint128{0, 1}, 128), 128},
	}
	for _, tt := range tests {
		if got := tt.a.prefixUnionLen(tt.b); got != tt.want {
			t.Errorf("%v.prefixUnion(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestBasicPrefixMap(t *testing.T) {
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
