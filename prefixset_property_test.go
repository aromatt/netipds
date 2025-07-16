package netipds

import (
	"math/rand"
	"net/netip"
	"testing"
)

// randomPrefixes returns n random prefixes of the specified IP version.
func randomPrefixes(rnd *rand.Rand, n int, ipv6 bool) []netip.Prefix {
	ps := make([]netip.Prefix, 0, n)
	for i := 0; i < n; i++ {
		var addr netip.Addr
		var maxPrefixLen int

		if ipv6 {
			var b [16]byte
			for j := 0; j < 16; j++ {
				b[j] = byte(rnd.Intn(256))
			}
			addr = netip.AddrFrom16(b)
			maxPrefixLen = 129
		} else {
			var b [4]byte
			for j := 0; j < 4; j++ {
				b[j] = byte(rnd.Intn(256))
			}
			addr = netip.AddrFrom4(b)
			maxPrefixLen = 33
		}

		plen := rnd.Intn(maxPrefixLen)
		ps = append(ps, netip.PrefixFrom(addr, plen).Masked())
	}
	return ps
}

// TestPrefixSetMergeRandom tests the merger of two large random PrefixSets.
func TestPrefixSetMergeRandom(t *testing.T) {
	tries := 100
	size := 1000
	tests := []struct {
		name string
		ipv6 bool
	}{
		{"IPv4", false},
		{"IPv6", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for seed := 0; seed < tries; seed++ {
				rnd := rand.New(rand.NewSource(int64(seed)))
				a := randomPrefixes(rnd, size, tt.ipv6)
				b := randomPrefixes(rnd, size, tt.ipv6)

				// Build random PrefixSets
				psbA, psbB := &PrefixSetBuilder{}, &PrefixSetBuilder{}
				for _, p := range a {
					if err := psbA.Add(p); err != nil {
						t.Fatalf("Add(a) failed: %v", err)
					}
				}
				for _, p := range b {
					if err := psbB.Add(p); err != nil {
						t.Fatalf("Add(b) failed: %v", err)
					}
				}

				psA := psbA.PrefixSet()
				psB := psbB.PrefixSet()

				// Create a fresh builder for the merge
				mergeBuilder := &PrefixSetBuilder{}
				for _, p := range psA.Prefixes() {
					if err := mergeBuilder.Add(p); err != nil {
						t.Fatalf("Add to merge builder failed: %v", err)
					}
				}
				mergeBuilder.Merge(psB)
				merged := mergeBuilder.PrefixSet().Prefixes()

				// Build expected set
				expMap := make(map[string]struct{})
				for _, p := range psA.Prefixes() {
					expMap[p.String()] = struct{}{}
				}
				for _, p := range psB.Prefixes() {
					expMap[p.String()] = struct{}{}
				}

				// Convert merged result to map for easy comparison
				mergedMap := make(map[string]struct{})
				for _, p := range merged {
					mergedMap[p.String()] = struct{}{}
				}

				// Check size of merged set
				if len(merged) != len(expMap) {
					t.Errorf("size mismatch - merged=%d, expected=%d",
						len(merged), len(expMap))
				}

				// Check every expected prefix is in merged
				for exp := range expMap {
					if _, ok := mergedMap[exp]; !ok {
						t.Errorf("missing prefix in merged: %s", exp)
					}
				}

				// Check no unexpected prefixes in merged
				for merged := range mergedMap {
					if _, ok := expMap[merged]; !ok {
						t.Errorf("unexpected prefix in merged: %s", merged)
					}
				}
			}
		})
	}
}
