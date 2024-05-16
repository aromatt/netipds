package netipmap

import (
	"testing"
)

var l = newLabel

func TestLabelIsBitZero(t *testing.T) {
	tests := []struct {
		a      label
		i      uint8
		want   bool
		wantOk bool
	}{
		{l(uint128{0, 0}, 128), 0, true, true},
		{l(uint128{0, 1}, 128), 0, true, true},
		{l(uint128{1 << 63, 0}, 128), 0, false, true},
		{l(uint128{1 << 62, 0}, 128), 1, false, true},
		{l(uint128{0, 1 << 63}, 128), 64, false, true},
		{l(uint128{0, 1}, 128), 127, false, true},
		{l(uint128{0, 2}, 128), 126, false, true},
		{l(uint128{^uint64(0), ^uint64(0)}, 128), 0, false, true},
		{l(uint128{^uint64(0), ^uint64(0)}, 128), 127, false, true},

		// i > l.len => false, false
		{l(uint128{1 << 63, 0}, 1), 1, false, false},
		{l(uint128{0, 0}, 128), 128, false, false},
	}
	for _, tt := range tests {
		if got, ok := tt.a.isBitZero(tt.i); got != tt.want || ok != tt.wantOk {
			t.Errorf("%v.isBitZero(%d) = (%v, %v), want (%v, %v)",
				tt.a, tt.i, got, ok, tt.want, tt.wantOk)
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

func TestParse(t *testing.T) {
	tests := []struct {
		s    string
		want label
	}{
		{"0/0", l(uint128{0, 0}, 0)},
		{"0/1", l(uint128{0, 0}, 1)},
		{"0/64", l(uint128{0, 0}, 64)},
		{"1/64", l(uint128{1, 0}, 64)},
		{"1/56", l(uint128{256, 0}, 56)},
		{"100/64", l(uint128{256, 0}, 64)},
		{"0/65", l(uint128{0, 0}, 65)},
		{"1/65", l(uint128{0, 1 << 63}, 65)},
		{"2/65", l(uint128{1, 0}, 65)},
		{"1/128", l(uint128{0, 1}, 128)},
		{"1/127", l(uint128{0, 2}, 127)},
		{"10000000000000001/128", l(uint128{1, 1}, 128)},
		{"100000000000001/120", l(uint128{1, 256}, 120)},
	}
	for _, tt := range tests {
		var got label
		if err := got.Parse(tt.s); err != nil {
			t.Errorf("label.Parse(%q) = %v", tt.s, err)
		} else if got != tt.want {
			t.Errorf("label.Parse(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestConcat(t *testing.T) {
	tests := []struct {
		a    label
		b    label
		want label
	}{
		{l(uint128{}, 0), l(uint128{}, 0), l(uint128{}, 0)},
		{l(uint128{}, 0), l(uint128{}, 1), l(uint128{}, 1)},
		{l(uint128{}, 0), l(uint128{0, 1}, 1), l(uint128{0, 1}, 1)},
		{l(uint128{}, 63), l(uint128{1 << 63, 0}, 1), l(uint128{1, 0}, 64)},
		{l(uint128{}, 127), l(uint128{1 << 63, 0}, 1), l(uint128{0, 1}, 128)},
		{l(uint128{1 << 63, 0}, 1), l(uint128{}, 1), l(uint128{1 << 63, 0}, 2)},
	}
	for _, tt := range tests {
		if got := tt.a.concat(tt.b); got != tt.want {
			t.Errorf("%v.concat(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
