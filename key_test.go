package netipds

import (
	"net/netip"
	"testing"
)

func k6(content uint128, offset, length uint8) key[keybits6] {
	return key[keybits6]{length, offset, content}
}

func k4(content uint32, offset, length uint8) key[keybits4] {
	return key[keybits4]{length, offset, keybits4(content)}
}

func TestKey4FromPrefix(t *testing.T) {
	tests := []struct {
		p    netip.Prefix
		want key[keybits4]
	}{
		{pfx("1.2.3.0/24"), k4(uint32(0x01020300), 0, 24)},
		{pfx("1.2.3.4/32"), k4(uint32(0x01020304), 0, 32)},
		{pfx("0.0.0.0/32"), k4(uint32(0), 0, 32)},
		{pfx("0.0.0.0/0"), k4(uint32(0), 0, 0)},
		{pfx("128.0.0.0/1"), k4(uint32(0x80000000), 0, 1)},
	}
	for _, tt := range tests {
		if got := key4FromPrefix(tt.p); got != tt.want {
			t.Errorf("key4FromPrefix(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestKey4ToPrefix(t *testing.T) {
	tests := []struct {
		k    key[keybits4]
		want netip.Prefix
	}{
		{k4(uint32(0x01020300), 0, 24), pfx("1.2.3.0/24")},
		{k4(uint32(0x01020304), 0, 32), pfx("1.2.3.4/32")},
		{k4(uint32(0), 0, 32), pfx("0.0.0.0/32")},
		{k4(uint32(0x80000000), 0, 1), pfx("128.0.0.0/1")},

		// zero key => default route
		{k4(uint32(0), 0, 0), pfx("0.0.0.0/0")},
	}
	for _, tt := range tests {
		if got := tt.k.ToPrefix(); got != tt.want {
			t.Errorf("%v.ToPrefix() = %v, want %v", tt.k, got, tt.want)
		}
	}
}

func TestKey4RoundTrip(t *testing.T) {
	tests := []struct {
		p netip.Prefix
	}{
		{pfx("0.0.0.0/32")},
		{pfx("1.2.3.0/24")},
		{pfx("1.2.3.4/32")},
		{pfx("128.0.0.0/1")},
		{pfx("0.0.0.0/0")},
	}
	for _, tt := range tests {
		if got := key4FromPrefix(tt.p).ToPrefix(); got != tt.p {
			t.Errorf("key4FromPrefix(%v).toPrefix() = %v, want %v", tt.p, got, tt.p)
		}
	}
}

func TestKey6FromPrefix(t *testing.T) {
	tests := []struct {
		p    netip.Prefix
		want key[keybits6]
	}{
		{pfx("::1/128"), k6(uint128{0, 1}, 0, 128)},
		{pfx("::2/127"), k6(uint128{0, 2}, 0, 127)},
		{pfx("8000::/1"), k6(uint128{1 << 63, 0}, 0, 1)},
		{pfx("::/0"), k6(uint128{}, 0, 0)},
	}
	for _, tt := range tests {
		if got := key6FromPrefix(tt.p); got != tt.want {
			t.Errorf("key6FromPrefix(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestKey6ToPrefix(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
		want netip.Prefix
	}{
		{k6(uint128{0, 1}, 0, 128), pfx("::1/128")},
		{k6(uint128{0, 2}, 0, 127), pfx("::2/127")},
		{k6(uint128{1 << 63, 0}, 0, 1), pfx("8000::/1")},

		// zero key => default route
		{k6(uint128{}, 0, 0), pfx("::/0")},
	}
	for _, tt := range tests {
		if got := tt.k.ToPrefix(); got != tt.want {
			t.Errorf("%v.ToPrefix() = %v, want %v", tt.k, got, tt.want)
		}
	}
}

func TestKey6RoundTrip(t *testing.T) {
	tests := []struct {
		p netip.Prefix
	}{
		{pfx("::0/128")},
		{pfx("::1/128")},
		{pfx("::2/127")},
		{pfx("::0/0")},
	}
	for _, tt := range tests {
		if got := key6FromPrefix(tt.p).ToPrefix(); got != tt.p {
			t.Errorf("key6FromPrefix(%v).toPrefix() = %v, want %v", tt.p, got, tt.p)
		}
	}
}

func TestKey4Bit(t *testing.T) {
	tests := []struct {
		p    netip.Prefix
		i    uint8
		want bit
	}{
		{pfx("0.0.0.0/32"), 0, bitL},
		{pfx("128.0.0.0/32"), 0, bitR},
		{pfx("0.0.0.0/32"), 31, bitL},
		{pfx("0.0.0.1/32"), 31, bitR},
		{pfx("0.0.0.1/32"), 32, bitL},
	}
	for _, tt := range tests {
		if got := key4FromPrefix(tt.p).Bit(tt.i); got != tt.want {
			t.Errorf("key4FromPrefix(%v).bit(%v) = %v, want %v", tt.p, tt.i, got, tt.want)
		}
	}

}

func TestKey6Bit(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
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

func TestKey4String(t *testing.T) {
	tests := []struct {
		k    key[keybits4]
		want string
	}{
		{k4(0, 0, 0), "0,0-0"},
		{k4(0, 0, 1), "0,0-1"},
		{k4(0x80000000, 0, 1), "1,0-1"},
		{k4(0x80000000, 0, 32), "80000000,0-32"},
		{k4(1, 0, 32), "1,0-32"},
		{k4(0x80000001, 0, 32), "80000001,0-32"},
		{k4(0x00800000, 0, 32), "800000,0-32"},
		{k4(0x80000000, 0, 8), "80,0-8"},
		{k4(0x00800000, 8, 16), "80,8-16"},
		{k4(0x00000001, 31, 32), "1,31-32"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestKey6String(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
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

func TestKey6IsPrefixOf(t *testing.T) {
	tests := []struct {
		a    netip.Prefix
		b    netip.Prefix
		want bool
	}{
		{pfx("::/0"), pfx("::/0"), true},
		{pfx("::/0"), pfx("::/1"), true},
		{pfx("::2/127"), pfx("::3/128"), true},
		{pfx("1::2/127"), pfx("1::3/128"), true},
		{pfx("1::/64"), pfx("1::1/128"), true},
		{pfx("8000::/1"), pfx("8000::1/128"), true},
		{pfx("8000::/1"), pfx("::1/128"), false},
	}
	for _, tt := range tests {
		if got := key6FromPrefix(tt.a).IsPrefixOf(key6FromPrefix(tt.b)); got != tt.want {
			t.Errorf("%v.isPrefixOf(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestKey6Truncated(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
		n    uint8
		want key[keybits6]
	}{
		{k6(uint128{0, 0}, 0, 0), 0, k6(uint128{0, 0}, 0, 0)},
		{k6(uint128{0, 0}, 0, 1), 1, k6(uint128{0, 0}, 0, 1)},
		{k6(uint128{0, 0}, 0, 1), 1, k6(uint128{0, 0}, 0, 1)},
		{k6(uint128{0, 1}, 0, 128), 1, k6(uint128{0, 0}, 0, 1)},

		{k6(uint128{0, 2}, 0, 127), 127, k6(uint128{0, 2}, 0, 127)},
		{k6(uint128{0, 2}, 0, 128), 127, k6(uint128{0, 2}, 0, 127)},
		{k6(uint128{0, 3}, 0, 128), 127, k6(uint128{0, 2}, 0, 127)},
	}
	for _, tt := range tests {
		if got := tt.k.Truncated(tt.n); got != tt.want {
			t.Errorf("%v.truncated(%d) = %v, want %v", tt.k, tt.n, got, tt.want)
		}
	}
}

func TestKey6Rooted(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
		want key[keybits6]
	}{
		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 0)},
		{k6(uint128{0, 1}, 64, 128), k6(uint128{0, 1}, 0, 128)},
		{k6(uint128{1, 0}, 64, 64), k6(uint128{1, 0}, 0, 64)},
	}
	for _, tt := range tests {
		if got := tt.k.Rooted(); got != tt.want {
			t.Errorf("%v.rooted() = %v, want %v", tt.k, got, tt.want)
		}
	}
}

func TestKey6Rest(t *testing.T) {
	tests := []struct {
		k    key[keybits6]
		i    uint8
		want key[keybits6]
	}{
		{k6(uint128{0, 0}, 0, 0), 0, k6(uint128{0, 0}, 0, 0)},
		{k6(uint128{1, 0}, 64, 64), 64, k6(uint128{1, 0}, 64, 64)},
		{k6(uint128{1, 0}, 64, 128), 127, k6(uint128{1, 0}, 127, 128)},
	}
	for _, tt := range tests {
		if got := tt.k.Rest(tt.i); got != tt.want {
			t.Errorf("%v.rest(%d) = %v, want %v", tt.k, tt.i, got, tt.want)
		}
	}
}

func TestKey6Next(t *testing.T) {
	tests := []struct {
		k         key[keybits6]
		wantLeft  key[keybits6]
		wantRight key[keybits6]
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
