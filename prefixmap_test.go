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
		{newLabel(uint128{0, 0}, 0), newLabel(uint128{0, 0}, 0), true},
		{newLabel(uint128{0, 0}, 0), newLabel(uint128{0, 0}, 1), true},
		{newLabel(uint128{0, 2}, 127), newLabel(uint128{0, 3}, 128), true},
		{newLabel(uint128{1, 2}, 127), newLabel(uint128{1, 3}, 128), true},
		{newLabel(uint128{1, 0}, 64), newLabel(uint128{1, 1}, 128), true},
		{newLabel(uint128{1 << 63, 0}, 1), newLabel(uint128{1 << 63, 1}, 128), true},
		{newLabel(uint128{1 << 63, 0}, 1), newLabel(uint128{0, 1}, 128), false},
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
		{newLabel(uint128{0, 0}, 0), newLabel(uint128{0, 0}, 0), 0},
		{newLabel(uint128{0, 0}, 0), newLabel(uint128{0, 0}, 1), 1},
		{newLabel(uint128{0, 0}, 1), newLabel(uint128{0, 0}, 0), 1},
		{newLabel(uint128{0, 0}, 1), newLabel(uint128{0, 0}, 1), 1},
		{newLabel(uint128{0, 0}, 1), newLabel(uint128{0, 0}, 2), 2},
		{newLabel(uint128{0, 0}, 2), newLabel(uint128{0, 0}, 1), 2},
		{newLabel(uint128{0, 0}, 2), newLabel(uint128{0, 0}, 2), 2},
		{newLabel(uint128{0, 0}, 127), newLabel(uint128{0, 1}, 128), 128},
	}
	for _, tt := range tests {
		if got := tt.a.prefixUnionLen(tt.b); got != tt.want {
			t.Errorf("%v.prefixUnion(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLabelString(t *testing.T) {
	tests := []struct {
		l    label
		want string
	}{
		{newLabel(uint128{0, 0}, 0), "0/0"},
		{newLabel(uint128{0, 0}, 1), "0/1"},
		{newLabel(uint128{0, 0}, 64), "0/64"},
		{newLabel(uint128{1, 0}, 64), "1/64"},
		{newLabel(uint128{256, 0}, 56), "1/56"},
		{newLabel(uint128{0, 0}, 65), "0/65"},
		{newLabel(uint128{0, 1 << 63}, 65), "1/65"},
		{newLabel(uint128{1, 0}, 65), "2/65"},
		{newLabel(uint128{0, 1}, 128), "1/128"},
		{newLabel(uint128{0, 2}, 127), "1/127"},
		{newLabel(uint128{1, 1}, 128), "10000000000000001/128"},
		{newLabel(uint128{1, 256}, 120), "100000000000001/120"},
	}
	for _, tt := range tests {
		if got := tt.l.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.l, got, tt.want)
		}
	}
}

func TestPrefixMapSetGet(t *testing.T) {
	tests := []struct {
		setPrefixes []string
		getPrefix   string
		want        bool
	}{
		//{[]string{}, "0::0/128", false},
		//{[]string{"0::0/128"}, "0::0/128", true},
		//{[]string{"0::1/128"}, "0::1/128", true},
		//{[]string{"0::2/128"}, "0::2/128", true},
		//{[]string{"0::2/127"}, "0::2/127", true},
		//{[]string{"1.2.3.0/24"}, "1.2.3.0/24", true},
		//{[]string{"1.2.3.0/24"}, "1.2.3.4/32", false},
		//{[]string{"0::0/128", "0::1/128", "0::2/127", "0::3/127"}, "0::1/128", true},
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
