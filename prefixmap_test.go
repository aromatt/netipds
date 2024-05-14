package netipmap

import (
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
