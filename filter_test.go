package netipds

import (
	"testing"
)

func TestFilterMightContain(t *testing.T) {
	tests := []struct {
		insert []key[keyBits6]
		query  key[keyBits6]
		want   bool
	}{
		{[]key[keyBits6]{k6(uint128{0, 1}, 0, 128)}, k6(uint128{0, 1}, 0, 128), true},
		{[]key[keyBits6]{k6(uint128{0, 1}, 0, 128)}, k6(uint128{0, 0}, 0, 128), false},

		{[]key[keyBits6]{k6(uint128{0, 2}, 0, 128)}, k6(uint128{0, 2}, 0, 128), true},
		{[]key[keyBits6]{k6(uint128{0, 1}, 0, 128)}, k6(uint128{0, 2}, 0, 128), false},

		{[]key[keyBits6]{k6(uint128{0, 3}, 0, 128)}, k6(uint128{0, 3}, 0, 128), true},
		{[]key[keyBits6]{k6(uint128{0, 3}, 0, 128)}, k6(uint128{0, 1}, 0, 128), false},

		{
			[]key[keyBits6]{
				k6(uint128{0, 2}, 0, 128),
				k6(uint128{0, 1}, 0, 128),
			},
			k6(uint128{0, 1}, 0, 128),
			true,
		},

		{
			[]key[keyBits6]{
				k6(uint128{0, 2}, 0, 128),
				k6(uint128{0, 1}, 0, 128),
			},
			k6(uint128{0, 3}, 0, 128),
			true,
		},

		{[]key[keyBits6]{k6(uint128{0, 2}, 0, 127)}, k6(uint128{0, 2}, 0, 127), true},
		{[]key[keyBits6]{k6(uint128{0, 2}, 0, 127)}, k6(uint128{0, 2}, 0, 128), false},
	}
	for _, tt := range tests {
		f := filter{}
		for _, k := range tt.insert {
			f.insert(k)
		}
		if got := f.mightContain(tt.query); got != tt.want {
			t.Errorf("f.mightContain(%v) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestFilterMightContainPrefix(t *testing.T) {
	tests := []struct {
		insert []key[keyBits6]
		query  key[keyBits6]
		want   bool
	}{
		{[]key[keyBits6]{k6(uint128{0, 1}, 0, 128)}, k6(uint128{0, 1}, 0, 128), true},
		{[]key[keyBits6]{k6(uint128{0, 1}, 0, 128)}, k6(uint128{0, 0}, 0, 128), false},

		{[]key[keyBits6]{k6(uint128{0, 0}, 0, 127)}, k6(uint128{0, 1}, 0, 128), true},
		{[]key[keyBits6]{k6(uint128{0, 2}, 0, 127)}, k6(uint128{0, 3}, 0, 128), true},
	}
	for _, tt := range tests {
		f := filter{}
		for _, k := range tt.insert {
			f.insert(k)
		}
		if got := f.mightContainPrefix(tt.query); got != tt.want {
			t.Errorf("f.mightContainPrefix(%v) = %v, want %v", tt.query, got, tt.want)
		}
	}
}
