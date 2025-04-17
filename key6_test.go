package netipds

import (
	"testing"
)

func k6(content uint128, offset, len uint8) key[keyBits6] {
	return key[keyBits6]{len, offset, content}
}

func TestKey6String(t *testing.T) {
	tests := []struct {
		k    key[keyBits6]
		want string
	}{
		{k6(uint128{0, 0}, 0, 0), "0,0-0"},
		{k6(uint128{0, 0}, 0, 1), "0,0-1"},
		{k6(uint128{0, 0}, 0, 64), "0,0-64"},
		{k6(uint128{1, 0}, 0, 64), "1,0-64"},
		{k6(uint128{256, 0}, 0, 56), "1,0-56"},
		{k6(uint128{256, 0}, 0, 64), "100,0-64"},
		{k6(uint128{0, 0}, 0, 65), "0,0-65"},
		{k6(uint128{0, 1 << 63}, 0, 65), "1,0-65"},
		{k6(uint128{1, 0}, 0, 65), "2,0-65"},
		{k6(uint128{0, 1}, 0, 128), "1,0-128"},
		{k6(uint128{0, 2}, 0, 127), "1,0-127"},
		{k6(uint128{1, 1}, 0, 128), "10000000000000001,0-128"},
		{k6(uint128{1, 256}, 0, 120), "100000000000001,0-120"},

		{k6(uint128{1<<63 + 1, 0}, 0, 64), "8000000000000001,0-64"},
		{k6(uint128{1<<63 + 1, 0}, 1, 64), "1,1-64"},
		{k6(uint128{1, 256}, 63, 120), "100000000000001,63-120"},
		{k6(uint128{1, 256}, 64, 120), "1,64-120"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestKey6Bit(t *testing.T) {
	tests := []struct {
		k    key[keyBits6]
		i    uint8
		want bit
	}{
		{k6(uint128{0, 0}, 0, 128), 0, bitL},
		{k6(uint128{0, 1}, 0, 128), 0, bitL},
		{k6(uint128{1 << 63, 0}, 0, 128), 0, bitR},
		{k6(uint128{1 << 62, 0}, 0, 128), 1, bitR},
		{k6(uint128{0, 1 << 63}, 0, 128), 64, bitR},
		{k6(uint128{0, 1}, 0, 128), 127, bitR},
		{k6(uint128{0, 2}, 0, 128), 126, bitR},
		{k6(uint128{^uint64(0), ^uint64(0)}, 0, 128), 0, bitR},
		{k6(uint128{^uint64(0), ^uint64(0)}, 0, 128), 127, bitR},
		// i > 127 => bitL
		{k6(uint128{^uint64(0), ^uint64(0)}, 0, 128), 128, bitL},
	}
	for _, tt := range tests {
		if got := tt.k.Bit(tt.i); got != tt.want {
			t.Errorf("%v.bit(%d) = %v, want %v",
				tt.k, tt.i, got, tt.want)
		}
	}
}

func TestKeyIsPrefixOf(t *testing.T) {
	tests := []struct {
		a    key[keyBits6]
		b    key[keyBits6]
		want bool
	}{
		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 0), true},
		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 1), true},
		{k6(uint128{0, 2}, 0, 127), k6(uint128{0, 3}, 0, 128), true},
		{k6(uint128{1, 2}, 0, 127), k6(uint128{1, 3}, 0, 128), true},
		{k6(uint128{1, 0}, 0, 64), k6(uint128{1, 1}, 0, 128), true},
		{k6(uint128{1 << 63, 0}, 0, 1), k6(uint128{1 << 63, 1}, 0, 128), true},
		{k6(uint128{1 << 63, 0}, 0, 1), k6(uint128{0, 1}, 0, 128), false},
	}
	for _, tt := range tests {
		if got := tt.a.IsPrefixOf(tt.b); got != tt.want {
			t.Errorf("%v.isPrefixOf(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestKeyNext(t *testing.T) {
	tests := []struct {
		k         key[keyBits6]
		wantLeft  key[keyBits6]
		wantRight key[keyBits6]
	}{
		{
			k:         k6(uint128{0, 0}, 0, 0),
			wantLeft:  k6(uint128{0, 0}, 0, 1),
			wantRight: k6(uint128{1 << 63, 0}, 0, 1),
		},
		{
			k:         k6(uint128{0, 0}, 0, 1),
			wantLeft:  k6(uint128{0, 0}, 1, 2),
			wantRight: k6(uint128{1 << 62, 0}, 1, 2),
		},
		{
			k:         k6(uint128{0, 2}, 0, 127),
			wantLeft:  k6(uint128{0, 2}, 127, 128),
			wantRight: k6(uint128{0, 3}, 127, 128),
		},
	}
	for _, tt := range tests {
		if got := tt.k.Next(bitL); got != tt.wantLeft {
			t.Errorf("%v.next(bitL) = %v, want %v", tt.k, got, tt.wantLeft)
		}
		if got := tt.k.Next(bitR); got != tt.wantRight {
			t.Errorf("%v.next(bitR) = %v, want %v", tt.k, got, tt.wantRight)
		}
	}
}
