package netipds

import (
	"fmt"
	"math/big"
	"net/netip"
	"testing"

	"go4.org/netipx"
	"pgregory.net/rapid"
)

const rapidSize = 500

// genIPv4Prefix yields a random IPv4 prefix (normalized to its network address).
func genIPv4Prefix(t *rapid.T) netip.Prefix {
	// generate 4 random bytes
	var raw [4]byte
	for i := range raw {
		raw[i] = byte(rapid.Byte().Draw(t, fmt.Sprintf("byte %d", i)))
	}
	// random mask length 0–32
	plen := int(rapid.IntRange(0, 32).Draw(t, "prefix length"))
	return netip.PrefixFrom(netip.AddrFrom4(raw), plen).Masked()
}

func genIPv6Prefix(t *rapid.T) netip.Prefix {
	// generate 16 random bytes
	var raw [16]byte
	for i := range raw {
		raw[i] = byte(rapid.Byte().Draw(t, fmt.Sprintf("byte %d", i)))
	}
	// random mask length 0–128
	plen := int(rapid.IntRange(0, 128).Draw(t, "prefix length"))
	return netip.PrefixFrom(netip.AddrFrom16(raw), plen).Masked()
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
	psA := builder(t, a)
	psA.Intersect(buildSet(t, b))
	return psA.PrefixSet()
}

func merge(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	t.Helper()
	psA := builder(t, a)
	psA.Merge(buildSet(t, b))
	return psA.PrefixSet()
}

func subtract(t *testing.T, a, b []netip.Prefix) *PrefixSet {
	t.Helper()
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

func assertSameIPs(t *testing.T, a, b *PrefixSet, msg string) {
	t.Helper()
}

// testPrefixSetMergeRandom tests the merger of two large random PrefixSets.
// It validates expected properties of merging:
// 1. Merging should be commutative: A + B = B + A
// 2. Merging with empty set should equal original set
// 3. Merging with self should equal self: A + A = A
// 4. Size of merged set should not exceed sum of sizes
// 5. Merged set should contain exactly the union of the operands
func testPrefixSetMergeRandom(t *testing.T, gen *rapid.Generator[netip.Prefix], size int) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, size, size).Draw(rt, "a")
		b := rapid.SliceOfN(gen, size, size).Draw(rt, "b")

		psA := buildSet(t, a)
		psB := buildSet(t, b)

		// Create merged sets
		psAB := merge(t, a, b)
		psBA := merge(t, b, a)

		// Property 1: Merging should be commutative: A + B = B + A
		assertSamePrefixes(t, psAB, psBA,
			"merging not commutative: A + B != B + A")

		// Property 2: Merging with empty set should equal original set
		psEmptyA := merge(t, a, []netip.Prefix{})
		if psEmptyA.Size() != psA.Size() {
			t.Errorf("Merging with empty set should equal original set, got size %d, expected %d",
				psEmptyA.Size(), psA.Size())
		}
		psEmptyB := merge(t, []netip.Prefix{}, b)
		if psEmptyB.Size() != psB.Size() {
			t.Errorf("Merging with empty set should equal original set, got size %d, expected %d",
				psEmptyB.Size(), psB.Size())
		}

		// Property 3: Merging with self should equal self: A + A = A
		assertSamePrefixes(t, merge(t, a, a), psA,
			"merging with self should equal self: A + A != A")
		assertSamePrefixes(t, merge(t, b, b), psB,
			"merging with self should equal self: B + B != B")

		// Property 4: Size of merged set should not exceed sum of sizes
		if psAB.Size() > psA.Size()+psB.Size() {
			t.Errorf("Merged size %d exceeds sum of sizes %d + %d",
				psAB.Size(), psA.Size(), psB.Size())
		}

		// Property 5: Merged set should contain exactly the union of the operands

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
	testPrefixSetMergeRandom(t, rapid.Custom(genIPv4Prefix), rapidSize)
}

func TestPrefixSetMergePropIPv6(t *testing.T) {
	testPrefixSetMergeRandom(t, rapid.Custom(genIPv6Prefix), rapidSize)
}

// testPrefixSetSubtractProp tests IP space subtraction of two random PrefixSets.
// This validates properties of IP address space subtraction (not set subtraction):
// 1. Subtracting empty set has no effect
// 2. Subtracting self removes all IPs
// 3. Result contains no IP addresses that are in B's IP space
// 4. Result contains only IP addresses that were in A's IP space
// 5. Should match netipx.IPSet behavior
func testPrefixSetSubtractProp(t *testing.T, gen *rapid.Generator[netip.Prefix], size int) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, size, size).Draw(rt, "a")
		b := rapid.SliceOfN(gen, size, size).Draw(rt, "b")
		psA := buildSet(t, a)
		psB := buildSet(t, b)

		// Create subtracted set A - B
		psSubtracted := subtract(t, a, b)

		// Property 1: Subtracting empty set preserves original IP space
		psAMinusEmpty := subtract(t, a, []netip.Prefix{})
		assertSamePrefixes(t, psSubtracted, psAMinusEmpty,
			"expected subtracting empty set to have no effect")

		// Property 2: Subtracting self removes all IP space
		if subtract(t, a, a).Size() != 0 {
			t.Errorf("Self subtraction should remove all IPs")
		}

		// Property 3: Result contains no IP addresses that are in B's IP space
		// Check this by ensuring no prefix in (A - B) is encompassed by B
		for _, p := range psSubtracted.Prefixes() {
			if psB.Encompasses(p) {
				t.Errorf("Subtracted result contains prefix %s that is encompassed by B", p)
			}
		}

		// Property 4: Result contains only IP addresses that were in A's IP space
		// Check this by ensuring every prefix in (A - B) is encompassed by A
		for _, p := range psSubtracted.Prefixes() {
			if !psA.Encompasses(p) {
				t.Errorf("Subtracted result contains prefix %s that is not encompassed by A", p)
			}
		}

		// Property 5: Should match netipx.IPSet behavior
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

		// Compare IP space coverage
		countExpected := countIPs(ipset.Prefixes())
		countSubtractedIPs := countIPs(psSubtracted.PrefixesCompact())

		if countExpected.Cmp(countSubtractedIPs) != 0 {
			t.Errorf("IP space mismatch - subtracted covers %s IPs, expected %s IPs",
				countSubtractedIPs.String(), countExpected.String())
		}
	})
}

func TestPrefixSetSubtractPropIPv4(t *testing.T) {
	testPrefixSetSubtractProp(t, rapid.Custom(genIPv4Prefix), rapidSize)
}

func TestPrefixSetSubtractPropIPv6(t *testing.T) {
	testPrefixSetSubtractProp(t, rapid.Custom(genIPv6Prefix), rapidSize)
}

// testPrefixSetIntersectRandom tests the intersection of two large random
// PrefixSets. It validates expected properties of intersection:
// 1. Intersection should be commutative: A & B = B & A
// 2. Intersection with empty set should be empty
// 3. Intersection with self should equal self: A & A = A
// 4. Size of intersection should not exceed size of either operand
// 5. Intersection should not contain any prefixes that are not encompassed by both sets
func testPrefixSetIntersectRandom(t *testing.T, gen *rapid.Generator[netip.Prefix], size int) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(gen, size, size).Draw(rt, "a")
		b := rapid.SliceOfN(gen, size, size).Draw(rt, "b")

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
	})
}

func TestPrefixSetIntersectPropIPv4(t *testing.T) {
	testPrefixSetIntersectRandom(t, rapid.Custom(genIPv4Prefix), rapidSize)
}

func TestPrefixSetIntersectPropIPv6(t *testing.T) {
	testPrefixSetIntersectRandom(t, rapid.Custom(genIPv6Prefix), rapidSize)
}

func TestPrefixSetContainsRapid(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		a := rapid.SliceOfN(rapid.Custom(genIPv4Prefix), rapidSize, rapidSize).Draw(rt, "a")
		psA := buildSet(t, a)
		for _, p := range a {
			if !psA.Contains(p) {
				t.Errorf("PrefixSet should contain %s", p)
			}
		}
	})
}
