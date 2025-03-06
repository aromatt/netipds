package netipds

import (
	"net/netip"
	"testing"
)

var k4 = NewKey4

func TestKey4FromPrefix(t *testing.T) {
	tests := []struct {
		p    netip.Prefix
		want key4
	}{
		{pfx("1.2.3.0/24"), k4(uint32(0x01020300), 0, 24)},
		// TODO add more
	}
	for _, tt := range tests {
		if got := key4FromPrefix(tt.p); got != tt.want {
			t.Errorf("key4FromPrefix(%v) = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestKey4Bit(t *testing.T) {
	tests := []struct {
		p    netip.Prefix
		bit  uint8
		want bit
	}{
		{pfx("0.0.0.0/32"), 0, bitL},
		{pfx("128.0.0.0/32"), 0, bitR},
		{pfx("0.0.0.0/32"), 31, bitL},
		{pfx("0.0.0.1/32"), 31, bitR},
	}
	for _, tt := range tests {
		if got := key4FromPrefix(tt.p).Bit(tt.bit); got != tt.want {
			t.Errorf("key4FromPrefix(%v).bit(%v) = %q, want %q", tt.p, tt.bit, got, tt.want)
		}
	}

}
