package netipds

// TODO
//import (
//	"testing"
//)

func k6(content uint128, offset, len uint8) key[keyBits6] {
	return key[keyBits6]{len, offset, keyBits6{content}}
}

//func TestKeyString(t *testing.T) {
//	tests := []struct {
//		k    key[keyBits6]
//		want string
//	}{
//		{k6(uint128{0, 0}, 0, 0), "0,0"},
//		{k6(uint128{0, 0}, 0, 1), "0,1"},
//		{k6(uint128{0, 0}, 0, 64), "0,64"},
//		{k6(uint128{1, 0}, 0, 64), "1,64"},
//		{k6(uint128{256, 0}, 0, 56), "1,56"},
//		{k6(uint128{256, 0}, 0, 64), "100,64"},
//		{k6(uint128{0, 0}, 0, 65), "0,65"},
//		{k6(uint128{0, 1 << 63}, 0, 65), "1,65"},
//		{k6(uint128{1, 0}, 0, 65), "2,65"},
//		{k6(uint128{0, 1}, 0, 128), "1,128"},
//		{k6(uint128{0, 2}, 0, 127), "1,127"},
//		{k6(uint128{1, 1}, 0, 128), "10000000000000001,128"},
//		{k6(uint128{1, 256}, 0, 120), "100000000000001,120"},
//
//		{k6(uint128{1<<63 + 1, 0}, 0, 64), "8000000000000001,64"},
//		{k6(uint128{1<<63 + 1, 0}, 1, 64), "8000000000000001,64"},
//		{k6(uint128{1, 256}, 63, 120), "100000000000001,120"},
//		{k6(uint128{1, 256}, 64, 120), "100000000000001,120"},
//	}
//	for _, tt := range tests {
//		if got := tt.k.content.String(); got != tt.want {
//			t.Errorf("%v.String() = %q, want %q", tt.k, got, tt.want)
//		}
//	}
//}

/* HACK
func TestKeyParse(t *testing.T) {
	tests := []struct {
		s    string
		want key
	}{
		{"0,0", k(uint128{0, 0}, 0, 0)},
		{"0,1", k(uint128{0, 0}, 0, 1)},
		{"0,64", k(uint128{0, 0}, 0, 64)},
		{"1,64", k(uint128{1, 0}, 0, 64)},
		{"1,56", k(uint128{256, 0}, 0, 56)},
		{"100,64", k(uint128{256, 0}, 0, 64)},
		{"0,65", k(uint128{0, 0}, 0, 65)},
		{"1,65", k(uint128{0, 1 << 63}, 0, 65)},
		{"2,65", k(uint128{1, 0}, 0, 65)},
		{"1,128", k(uint128{0, 1}, 0, 128)},
		{"1,127", k(uint128{0, 2}, 0, 127)},
		{"10000000000000001,128", k(uint128{1, 1}, 0, 128)},
		{"100000000000001,120", k(uint128{1, 256}, 0, 120)},

		{"8000000000000001,64", k(uint128{1<<63 + 1, 0}, 0, 64)},
		{"1,64", k(uint128{1, 0}, 0, 64)},
		{"100000000000001,120", k(uint128{1, 256}, 0, 120)},
		{"1,120", k(uint128{0, 256}, 0, 120)},
	}
	for _, tt := range tests {
		var got key
		if err := got.Parse(tt.s); err != nil {
			t.Errorf("key.Parse(%q) = %v", tt.s, err)
		} else if got != tt.want {
			t.Errorf("key.Parse(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}
*/

//func TestKeyBit(t *testing.T) {
//	tests := []struct {
//		k    key6
//		i    uint8
//		want uint8
//	}{
//		{k6(uint128{0, 0}, 0, 128), 0, 0},
//		{k6(uint128{0, 1}, 0, 128), 0, 0},
//		{k6(uint128{1 << 63, 0}, 0, 128), 0, 1},
//		{k6(uint128{1 << 62, 0}, 0, 128), 1, 1},
//		{k6(uint128{0, 1 << 63}, 0, 128), 64, 1},
//		{k6(uint128{0, 1}, 0, 128), 127, 1},
//		{k6(uint128{0, 2}, 0, 128), 126, 1},
//		{k6(uint128{^uint64(0), ^uint64(0)}, 0, 128), 0, 1},
//		{k6(uint128{^uint64(0), ^uint64(0)}, 0, 128), 127, 1},
//	}
//	for _, tt := range tests {
//		if got := tt.k.Bit(tt.i); got != tt.want {
//			t.Errorf("%v.bit(%d) = %v, want %v",
//				tt.k, tt.i, got, tt.want)
//		}
//	}
//}
//
//func TestKeyIsPrefixOf(t *testing.T) {
//	tests := []struct {
//		a    key6
//		b    key6
//		want bool
//	}{
//		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 0), true},
//		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 1), true},
//		{k6(uint128{0, 2}, 0, 127), k6(uint128{0, 3}, 0, 128), true},
//		{k6(uint128{1, 2}, 0, 127), k6(uint128{1, 3}, 0, 128), true},
//		{k6(uint128{1, 0}, 0, 64), k6(uint128{1, 1}, 0, 128), true},
//		{k6(uint128{1 << 63, 0}, 0, 1), k6(uint128{1 << 63, 1}, 0, 128), true},
//		{k6(uint128{1 << 63, 0}, 0, 1), k6(uint128{0, 1}, 0, 128), false},
//	}
//	for _, tt := range tests {
//		if got := tt.a.IsPrefixOf(tt.b, false); got != tt.want {
//			t.Errorf("%v.isPrefixOf(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
//		}
//	}
//}
//
//func TestKeyNext(t *testing.T) {
//	tests := []struct {
//		k         key6
//		wantLeft  key6
//		wantRight key6
//	}{
//		{k6(uint128{0, 0}, 0, 0), k6(uint128{0, 0}, 0, 1), k6(uint128{1 << 63, 0}, 0, 1)},
//		{k6(uint128{0, 0}, 0, 1), k6(uint128{0, 0}, 1, 2), k6(uint128{1 << 62, 0}, 1, 2)},
//		{k6(uint128{0, 2}, 0, 127), k6(uint128{0, 2}, 127, 128), k6(uint128{0, 3}, 127, 128)},
//	}
//	for _, tt := range tests {
//		if got := tt.k.Next(bitL); got != tt.wantLeft {
//			t.Errorf("%v.next(bitL) = %v, want %v", tt.k, got, tt.wantLeft)
//		}
//		if got := tt.k.Next(bitR); got != tt.wantRight {
//			t.Errorf("%v.next(bitR) = %v, want %v", tt.k, got, tt.wantRight)
//		}
//	}
//}
