package netipmap

import (
	"testing"
)

var l = newLabel

func TestGetBit(t *testing.T) {
	tests := []struct {
		a    label
		i    uint8
		want bool
	}{
		{l(uint128{0, 0}, 128), 0, false},
		{l(uint128{0, 1}, 128), 0, false},
		{l(uint128{1 << 63, 0}, 128), 0, true},
		{l(uint128{1 << 62, 0}, 128), 1, true},
		{l(uint128{0, 1 << 63}, 128), 64, true},
		{l(uint128{0, 1}, 128), 127, true},
		{l(uint128{0, 2}, 128), 126, true},
		{l(uint128{^uint64(0), ^uint64(0)}, 128), 0, true},
		{l(uint128{^uint64(0), ^uint64(0)}, 128), 127, true},
		{l(uint128{^uint64(0), ^uint64(0)}, 128), 128, false},
	}
	for _, tt := range tests {
		if got := tt.a.getBit(tt.i); got != tt.want {
			t.Errorf("%v.getBit(%d) = %v, want %v", tt.a, tt.i, got, tt.want)
		}
	}
}

func TestIsPrefixOf(t *testing.T) {
	tests := []struct {
		a    label
		b    label
		want bool
	}{
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 0), true},
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 1), true},
		{l(uint128{0, 2}, 127), l(uint128{0, 3}, 128), true},
		{l(uint128{1, 2}, 127), l(uint128{1, 3}, 128), true},
		{l(uint128{1, 0}, 64), l(uint128{1, 1}, 128), true},
		{l(uint128{1 << 63, 0}, 1), l(uint128{1 << 63, 1}, 128), true},
		{l(uint128{1 << 63, 0}, 1), l(uint128{0, 1}, 128), false},
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
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 0), 0},
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 1), 1},
		{l(uint128{0, 0}, 1), l(uint128{0, 0}, 0), 1},
		{l(uint128{0, 0}, 1), l(uint128{0, 0}, 1), 1},
		{l(uint128{0, 0}, 1), l(uint128{0, 0}, 2), 2},
		{l(uint128{0, 0}, 2), l(uint128{0, 0}, 1), 2},
		{l(uint128{0, 0}, 2), l(uint128{0, 0}, 2), 2},
		{l(uint128{0, 0}, 127), l(uint128{0, 1}, 128), 128},
	}
	for _, tt := range tests {
		if got := tt.a.prefixUnionLen(tt.b); got != tt.want {
			t.Errorf("%v.prefixUnion(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		l    label
		want string
	}{
		{l(uint128{0, 0}, 0), "0/0"},
		{l(uint128{0, 0}, 1), "0/1"},
		{l(uint128{0, 0}, 64), "0/64"},
		{l(uint128{1, 0}, 64), "1/64"},
		{l(uint128{256, 0}, 56), "1/56"},
		{l(uint128{256, 0}, 64), "100/64"},
		{l(uint128{0, 0}, 65), "0/65"},
		{l(uint128{0, 1 << 63}, 65), "1/65"},
		{l(uint128{1, 0}, 65), "2/65"},
		{l(uint128{0, 1}, 128), "1/128"},
		{l(uint128{0, 2}, 127), "1/127"},
		{l(uint128{1, 1}, 128), "10000000000000001/128"},
		{l(uint128{1, 256}, 120), "100000000000001/120"},
	}
	for _, tt := range tests {
		if got := tt.l.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.l, got, tt.want)
		}
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		s    string
		want label
	}{
		//{"0/0", l(uint128{0, 0}, 0)},
		//{"0/1", l(uint128{0, 0}, 1)},
		//{"0/64", l(uint128{0, 0}, 64)},
		//{"1/64", l(uint128{1, 0}, 64)},
		//{"2/65", l(uint128{1, 0}, 65)},
	}
	for _, tt := range tests {
		if got, err := labelFromString(tt.s); err != nil || got != tt.want {
			t.Errorf("labelFromString(%q) = %v, %v, want %v, nil", tt.s, got, err, tt.want)
		}
	}
}

func TestConcat(t *testing.T) {
	tests := []struct {
		a    label
		b    label
		want label
	}{
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 0), l(uint128{0, 0}, 0)},
		{l(uint128{0, 0}, 0), l(uint128{0, 0}, 1), l(uint128{0, 0}, 1)},
		{l(uint128{0, 0}, 0), l(uint128{0, 1}, 1), l(uint128{0, 1}, 1)},
		{l(uint128{0, 0}, 63), l(uint128{1 << 63, 0}, 1), l(uint128{1, 0}, 64)},
		{l(uint128{0, 0}, 127), l(uint128{1 << 63, 0}, 1), l(uint128{0, 1}, 128)},
	}
	for _, tt := range tests {
		if got := tt.a.concat(tt.b); got != tt.want {
			t.Errorf("%v.concat(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
