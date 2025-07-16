package netipds

import (
	"math/big"
	"math/rand"
	"net/netip"
	"testing"

	"go4.org/netipx"
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

// countIPs returns the total number of IP addresses covered by the prefixes
func countIPs(prefixes []netip.Prefix) *big.Int {
	total := big.NewInt(0)
	for _, p := range prefixes {
		if p.Addr().Is4() {
			// IPv4: 2^(32-prefixLen) addresses
			hostBits := 32 - p.Bits()
			count := big.NewInt(1)
			count.Lsh(count, uint(hostBits))
			total.Add(total, count)
		} else {
			// IPv6: 2^(128-prefixLen) addresses
			hostBits := 128 - p.Bits()
			count := big.NewInt(1)
			count.Lsh(count, uint(hostBits))
			total.Add(total, count)
		}
	}
	return total
}

func builder(t *testing.T, ps []netip.Prefix) *PrefixSetBuilder {
	b := &PrefixSetBuilder{}
	for _, p := range ps {
		if err := b.Add(p); err != nil {
			t.Fatalf("buildSet: Add(%v) failed: %v", p, err)
		}
	}
	return b
}

func buildSet(t *testing.T, ps []netip.Prefix) *PrefixSet {
	return builder(t, ps).PrefixSet()
}

func intersect(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	psA := builder(t, a)
	psA.Intersect(buildSet(t, b))
	return psA.PrefixSet()
}

func merge(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	psA := builder(t, a)
	psA.Merge(buildSet(t, b))
	return psA.PrefixSet()
}

func subtract(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	psA := builder(t, a)
	psA.Subtract(buildSet(t, b))
	return psA.PrefixSet()
}

// assertSamePrefixesSlices fails if the two PrefixSets do not have the same
// Prefixes() result.
func assertSamePrefixes(t *testing.T, a, b *PrefixSet, msg string) {
	t.Helper()
	bPrefixes := b.Prefixes()
	for i, p := range a.Prefixes() {
		if p != bPrefixes[i] {
			t.Fatalf("prefix slices differ; " + msg)
		}
	}
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

				psA := buildSet(t, a)
				psB := buildSet(t, b)

				// Create a fresh builder for the merge
				psAB := merge(t, a, b)
				merged := psAB.Prefixes()

				// Build expected set as simple union (preserving all individual prefixes)
				expected := make(map[string]struct{})
				for _, p := range psA.Prefixes() {
					expected[p.String()] = struct{}{}
				}
				for _, p := range psB.Prefixes() {
					expected[p.String()] = struct{}{}
				}

				// Convert merged result to map for easy comparison
				mergedMap := make(map[string]struct{})
				for _, p := range merged {
					mergedMap[p.String()] = struct{}{}
				}

				// Check size of merged set
				if len(merged) != len(expected) {
					t.Errorf("size mismatch - merged=%d, expected=%d",
						len(merged), len(expected))
				}

				// Check every expected prefix is in merged
				for exp := range expected {
					if _, ok := mergedMap[exp]; !ok {
						t.Errorf("missing prefix in merged: %s", exp)
					}
				}

				// Check no unexpected prefixes in merged
				for merged := range mergedMap {
					if _, ok := expected[merged]; !ok {
						t.Errorf("unexpected prefix in merged: %s", merged)
					}
				}
			}
		})
	}
}

// TestPrefixSetSubtractRandom tests the subtraction of two large random PrefixSets.
func TestPrefixSetSubtractRandom(t *testing.T) {
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

				psA := buildSet(t, a)
				psB := buildSet(t, b)

				// Create a fresh builder for the subtraction
				subtracted := subtract(t, a, b).Prefixes()

				// Build expected set using netipx.IPSet as oracle
				var ipsb netipx.IPSetBuilder
				for _, p := range psA.Prefixes() {
					ipsb.AddPrefix(p)
				}
				for _, p := range psB.Prefixes() {
					ipsb.RemovePrefix(p)
				}
				ipset, err := ipsb.IPSet()
				if err != nil {
					t.Fatalf("Oracle IPSet build failed: %v", err)
				}
				expected := make(map[string]struct{})
				for _, p := range ipset.Prefixes() {
					expected[p.String()] = struct{}{}
				}

				// Convert subtracted result to map for easy comparison
				subtractedMap := make(map[string]struct{})
				for _, p := range subtracted {
					subtractedMap[p.String()] = struct{}{}
				}

				// Check size of subtracted set
				if len(subtracted) != len(expected) {
					t.Errorf("size mismatch - subtracted=%d, expected=%d",
						len(subtracted), len(expected))
				}

				// Check every expected prefix is in subtracted
				for exp := range expected {
					if _, ok := subtractedMap[exp]; !ok {
						t.Errorf("missing prefix in subtracted: %s", exp)
					}
				}

				// Check no unexpected prefixes in subtracted
				for sub := range subtractedMap {
					if _, ok := expected[sub]; !ok {
						t.Errorf("unexpected prefix in subtracted: %s", sub)
					}
				}
			}
		})
	}
}

// TestPrefixSetIntersectRandom tests the intersection of two large random
// PrefixSets. It validates expected properties of intersection:
// 1. Intersection should be commutative: A & B = B & A
// 2. Intersection with empty set should be empty
// 3. Intersection with self should equal self: A & A = A
// 4. Size of intersection should not exceed size of either operand
// 5. Intersection should not contain any prefixes that are not encompassed by both sets
func TestPrefixSetIntersectRandom(t *testing.T) {
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

				psA := buildSet(t, a)
				psB := buildSet(t, b)
				psAB := intersect(t, a, b)
				psBA := intersect(t, b, a)

				// Property 1: Intersection should be commutative: A & B = B & A
				assertSamePrefixes(t, psAB, psBA,
					"intersection not commutative: A & B != B & A")

				// Property 2: Intersection with empty set should be empty
				psEmpty := intersect(t, a, []netip.Prefix{})
				if psEmpty.Size() != 0 {
					t.Errorf("Intersection with empty set should be empty, got size %d",
						psEmpty.Size())
				}

				// Property 3: Intersection with self should equal self
				assertSamePrefixes(t, intersect(t, a, a), psA,
					"intersection with self should equal self: A & A != A")

				// Property 4: IP address count of intersection should not
				// exceed that of either operand
				countA := countIPs(psA.PrefixesCompact())
				countB := countIPs(psB.PrefixesCompact())
				countInter := countIPs(psAB.PrefixesCompact())
				if countInter.Cmp(countA) > 0 {
					t.Errorf("Intersection IP count %s exceeds first operand IP count %s",
						countInter.String(), countA.String())
				}
				if countInter.Cmp(countB) > 0 {
					t.Errorf("Intersection IP count %s exceeds second operand IP count %s",
						countInter.String(), countB.String())
				}

				// Property 5: Intersection should not contain any prefixes
				// that are not encompassed by both sets
				for _, p := range psAB.PrefixesCompact() {
					if !psA.Encompasses(p) || !psB.Encompasses(p) {
						t.Errorf("Prefix %s in intersection not encompassed by both sets", p)
					}
				}
			}
		})
	}
}
