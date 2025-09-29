package netipds

import (
	"fmt"
	"math/big"
	"net/netip"
	"testing"

	"go4.org/netipx"
	"pgregory.net/rapid"
)

// genPrefixLen uses rapid.IntRange to generate a random prefix length, biased
// toward small networks and biased against /0.
//
// rapid.IntRange is biased toward small values and the max value. But small
// prefixes == large networks, which collapses the explored input space down to
// degenerate edge cases. For example:
//   - If a PrefixSetBuilder contains /0, then PrefixSetBuilder.Intersect(b)
//     will always include all of b.
//   - If a PrefixSet contains /0, then subtracting it from a PrefixSetBuilder
//     will always result in an empty set.
func genPrefixLen(t *rapid.T, maxLen int) int {
	val := rapid.IntRange(0, maxLen).Filter(func(v int) bool {
		if v < maxLen {
			return true
		}
		// 5 because ~0 and 10 are over-represented
		return rapid.IntRange(0, 10).Draw(t, "roll for /0") == 5
	}).Draw(t, "prefix len")
	return maxLen - val
}

func genIPv4Prefix(t *rapid.T) netip.Prefix {
	var raw [4]byte
	for i := range raw {
		raw[i] = byte(rapid.Byte().Draw(t, fmt.Sprintf("byte %d", i)))
	}
	return netip.PrefixFrom(netip.AddrFrom4(raw), genPrefixLen(t, 32)).Masked()
}

func genIPv6Prefix(t *rapid.T) netip.Prefix {
	var raw [16]byte
	for i := range raw {
		raw[i] = byte(rapid.Byte().Draw(t, fmt.Sprintf("byte %d", i)))
	}
	return netip.PrefixFrom(netip.AddrFrom16(raw), genPrefixLen(t, 128)).Masked()
}

// countIPs returns the total number of IP addresses covered by a PrefixSet.
//
// Note: this depends on the correctness of PrefixSet.PrefixesCompact.
func countIPs(s *PrefixSet) *big.Int {
	total := big.NewInt(0)
	for _, p := range s.PrefixesCompact() {
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
	t.Helper()
	b := &PrefixSetBuilder{}
	for _, p := range ps {
		if err := b.Add(p); err != nil {
			t.Fatalf("buildSet: Add(%v) failed: %v", p, err)
		}
	}
	return b
}

func buildSet(t *testing.T, ps []netip.Prefix) *PrefixSet {
	t.Helper()
	return builder(t, ps).PrefixSet()
}

func intersect(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	t.Helper()
	psbA := builder(t, a)
	psbA.Intersect(buildSet(t, b))
	return psbA.PrefixSet()
}

func merge(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	t.Helper()
	psbA := builder(t, a)
	psbA.Merge(buildSet(t, b))
	return psbA.PrefixSet()
}

func subtract(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	t.Helper()
	psbA := builder(t, a)
	psbA.Subtract(buildSet(t, b))
	return psbA.PrefixSet()
}

// ipsetSubtract subtracts the IP space of PrefixSet b from PrefixSet a
// using netipx.IPSetBuilder. It returns a new netipx.IPSet containing the
// result of the subtraction.
func ipsetSubtract(a, b *PrefixSet) (*netipx.IPSet, error) {
	var ipsb netipx.IPSetBuilder
	for _, p := range a.Prefixes() {
		ipsb.AddPrefix(p)
	}
	for _, p := range b.Prefixes() {
		ipsb.RemovePrefix(p)
	}
	return ipsb.IPSet()
}

// assertSamePrefixesSlices fails if the two PrefixSets do not have the same
// Prefixes() result.
func assertSamePrefixes(t *testing.T, a, b *PrefixSet, msg string) {
	t.Helper()
	aPrefixes := a.Prefixes()
	bPrefixes := b.Prefixes()
	if len(aPrefixes) != len(bPrefixes) {
		t.Fatalf("prefix slices differ in length: |a| = %d, |b| = %d; %s",
			len(aPrefixes), len(bPrefixes), msg)
	}
	for i, p := range a.Prefixes() {
		if p != bPrefixes[i] {
			t.Fatalf("prefix slices differ; " + msg)
		}
	}
}

// prefixSetToIPset converts a PrefixSet to a netipx.IPSet.
func prefixSetToIPset(ps *PrefixSet) *netipx.IPSet {
	ipsb := &netipx.IPSetBuilder{}
	for _, p := range ps.Prefixes() {
		ipsb.AddPrefix(p)
	}
	ipset, err := ipsb.IPSet()
	if err != nil {
		panic(fmt.Sprintf("prefixSetToIPset: IPSet() failed: %v", err))
	}
	return ipset
}

// testPrefixSetMergeRandom tests the merger of two large random PrefixSets.
// It validates expected properties of merging:
//  1. Merging should be commutative: A + B = B + A
//  2. Merging with empty set should equal original set
//  3. Merging with self should equal self: A + A = A
//  4. Merged set should contain exactly the union of the operands
func testPrefixSetMergeRandom(t *testing.T, gen *rapid.Generator[netip.Prefix]) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, 10, 1000).Draw(rt, "a")
		b := rapid.SliceOfN(gen, 10, 1000).Draw(rt, "b")

		psA := buildSet(t, a)
		psB := buildSet(t, b)

		// Create merged sets
		psAB := merge(t, a, b)
		psBA := merge(t, b, a)

		// Property 1: Merging should be commutative: A + B = B + A
		assertSamePrefixes(t, psAB, psBA,
			"merging not commutative: A + B != B + A")

		// Property 2: Merging with empty set should equal original set
		msg := "expected merging with empty set to have no effect"
		assertSamePrefixes(t, psA, merge(t, a, []netip.Prefix{}), msg)
		assertSamePrefixes(t, psA, merge(t, []netip.Prefix{}, a), msg)
		assertSamePrefixes(t, psB, merge(t, b, []netip.Prefix{}), msg)
		assertSamePrefixes(t, psB, merge(t, []netip.Prefix{}, b), msg)

		// Property 3: Merging with self should equal self: A + A = A
		assertSamePrefixes(t, merge(t, a, a), psA,
			"merging with self should equal self: A + A != A")
		assertSamePrefixes(t, merge(t, b, b), psB,
			"merging with self should equal self: B + B != B")

		// Property 4: Merged set should contain exactly the union of the operands

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
		mergedAB := psAB.Prefixes()
		for _, p := range mergedAB {
			mergedMap[p.String()] = struct{}{}
		}

		// Check size of merged set
		if len(mergedAB) != len(expected) {
			t.Errorf("size mismatch - merged=%d, expected=%d",
				len(mergedAB), len(expected))
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
	})
}

func TestPrefixSetMergePropIPv4(t *testing.T) {
	testPrefixSetMergeRandom(t, rapid.Custom(genIPv4Prefix))
}

func TestPrefixSetMergePropIPv6(t *testing.T) {
	testPrefixSetMergeRandom(t, rapid.Custom(genIPv6Prefix))
}

// testPrefixSetSubtractProp tests IP space subtraction of two random PrefixSets.
// This validates properties of IP address space subtraction (not set subtraction):
//  1. Subtracting empty set has no effect
//  2. Subtracting self removes all IPs
//  3. Result contains no IP addresses that are in B's IP space
//  4. Result contains only IP addresses that were in A's IP space
//  5. Result has same IP space coverage as netipx.IPSet implementation
func testPrefixSetSubtractProp(t *testing.T, gen *rapid.Generator[netip.Prefix]) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, 1, 1000).Draw(rt, "a")
		b := rapid.SliceOfN(gen, 1, 1000).Draw(rt, "b")
		psA := buildSet(t, a)
		psB := buildSet(t, b)

		// Create subtracted set A - B
		psSubtracted := subtract(t, a, b)

		// Property 1: Subtracting empty set preserves original IP space
		psAMinusEmpty := subtract(t, a, []netip.Prefix{})
		assertSamePrefixes(t, psA, psAMinusEmpty,
			"expected subtracting empty set to have no effect")

		// Property 2: Subtracting self removes all IP space
		if subtract(t, a, a).Size() != 0 {
			t.Errorf("Self subtraction should remove all IPs")
		}

		// Property 3: Result contains no IP addresses that are in B's IP space.
		// Check this by ensuring no prefix in (A - B) is encompassed by B.
		for _, p := range psSubtracted.Prefixes() {
			if psB.Encompasses(p) {
				t.Errorf("Subtracted result contains prefix %s that is encompassed by B", p)
			}
		}

		// Property 4: Result contains only IP addresses that were in A's IP space.
		// Check this by ensuring every prefix in (A - B) is encompassed by A.
		for _, p := range psSubtracted.Prefixes() {
			if !psA.Encompasses(p) {
				t.Errorf("Subtracted result contains prefix %s that is not encompassed by A", p)
			}
		}

		// Property 5: Result has same IP space coverage as netipx.IPSet implementation
		ipset, err := ipsetSubtract(psA, psB)
		if err != nil {
			t.Fatalf("Oracle IPSet build failed: %v", err)
		}

		// Compare IP space coverage
		netipdsIpSet := prefixSetToIPset(psSubtracted)
		if !netipdsIpSet.Equal(ipset) {
			t.Errorf("IP space mismatch against netipx implementation:\nA: %v\nB: %v\nExpected: %v\nActual: %v",
				a, b, ipset.Prefixes(), netipdsIpSet.Prefixes())
		}
	})
}

func TestPrefixSetSubtractPropIPv4(t *testing.T) {
	testPrefixSetSubtractProp(t, rapid.Custom(genIPv4Prefix))
}

func TestPrefixSetSubtractPropIPv6(t *testing.T) {
	testPrefixSetSubtractProp(t, rapid.Custom(genIPv6Prefix))
}

// testPrefixSetIntersectProp tests the intersection of two large random
// PrefixSets. It validates expected properties of intersection:
//  1. Intersection should be commutative: A & B = B & A
//  2. Intersection with empty set should be empty
//  3. Intersection with self should equal self: A & A = A
//  4. Size of intersection should not exceed size of either operand
//  5. Intersection should not contain any prefixes that are not encompassed by both sets
func testPrefixSetIntersectProp(t *testing.T, gen *rapid.Generator[netip.Prefix]) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, 10, 1000).Draw(rt, "a")
		b := rapid.SliceOfN(gen, 10, 1000).Draw(rt, "b")

		// Build intersection and prefix set separately; even though a single builder
		// could be used for both, that property is not under test here.
		psA := buildSet(t, a)
		psB := buildSet(t, b)
		psAB := intersect(t, a, b)
		psBA := intersect(t, b, a)

		// Property 1: Intersection should be commutative: A & B = B & A
		if psAB.Size() != psBA.Size() {
			t.Errorf("Intersection not commutative: A & B has %d prefixes, B & A has %d prefixes",
				psAB.Size(), psBA.Size())
			t.Errorf("A: %v", a)
			t.Errorf("B: %v", b)
			t.Errorf("A & B: %v", psAB.Prefixes())
			t.Errorf("B & A: %v", psBA.Prefixes())
		}
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
		countA := countIPs(psA)
		countB := countIPs(psB)
		countInter := countIPs(psAB)
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
	})
}

func TestPrefixSetIntersectPropIPv4(t *testing.T) {
	testPrefixSetIntersectProp(t, rapid.Custom(genIPv4Prefix))
}

func TestPrefixSetIntersectPropIPv6(t *testing.T) {
	testPrefixSetIntersectProp(t, rapid.Custom(genIPv6Prefix))
}

func TestPrefixSetContainsProp(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(rapid.Custom(genIPv4Prefix), 10, 1000).Draw(rt, "a")
		psA := buildSet(t, a)
		for _, p := range a {
			if !psA.Contains(p) {
				t.Errorf("PrefixSet should contain %s", p)
			}
		}
	})
}
