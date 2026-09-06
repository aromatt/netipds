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

// mustIPSet builds the netipx coverage oracle for prefixes and fails the
// property immediately if netipx rejects the generated input.
func mustIPSet(t *rapid.T, prefixes []netip.Prefix) *netipx.IPSet {
	var b netipx.IPSetBuilder
	for _, p := range prefixes {
		b.AddPrefix(p)
	}
	result, err := b.IPSet()
	if err != nil {
		t.Fatalf("building oracle IPSet: %v", err)
	}
	return result
}

// mutateIPSet returns a copy of current after applying mutate through a new
// netipx builder. Keeping the oracle immutable mirrors PrefixSet snapshots.
func mutateIPSet(t *rapid.T, current *netipx.IPSet, mutate func(*netipx.IPSetBuilder)) *netipx.IPSet {
	var b netipx.IPSetBuilder
	b.AddSet(current)
	mutate(&b)
	result, err := b.IPSet()
	if err != nil {
		t.Fatalf("mutating oracle IPSet: %v", err)
	}
	return result
}

// siblingPrefix returns the prefix sharing a parent node with p: the same
// length with its final bit flipped. A /0 has no sibling, so it is returned
// unchanged.
//
// Ancestors and descendants of a prefix all lie on a single path from the
// root, so a generator that only relates prefixes that way never produces the
// branch nodes that hold two children.
func siblingPrefix(p netip.Prefix) netip.Prefix {
	if p.Bits() == 0 {
		return p
	}
	last := p.Bits() - 1
	flip := byte(1) << (7 - last%8)
	if p.Addr().Is4() {
		raw := p.Addr().As4()
		raw[last/8] ^= flip
		return netip.PrefixFrom(netip.AddrFrom4(raw), p.Bits())
	}
	raw := p.Addr().As16()
	raw[last/8] ^= flip
	return netip.PrefixFrom(netip.AddrFrom16(raw), p.Bits())
}

func prefixMaxBits(p netip.Prefix) int {
	if p.Addr().Is4() {
		return 32
	}
	return 128
}

// copyPrefixBits replaces the first bits bits of dst with the corresponding
// bits from src, preserving every later bit in dst.
func copyPrefixBits(dst, src []byte, bits int) {
	wholeBytes := bits / 8
	copy(dst[:wholeBytes], src[:wholeBytes])
	if remaining := bits % 8; remaining != 0 {
		mask := byte(0xff << (8 - remaining))
		dst[wholeBytes] = dst[wholeBytes]&^mask | src[wholeBytes]&mask
	}
}

// flipAddrBit toggles the named network-order bit in raw.
func flipAddrBit(raw []byte, bit int) {
	raw[bit/8] ^= byte(1) << (7 - bit%8)
}

// drawRelatedAddr generates an address in base's family. Its first keepBits
// match base; when flipBit is non-negative, that bit is forced to differ.
func drawRelatedAddr(t *rapid.T, base netip.Prefix, keepBits, flipBit int, label string) netip.Addr {
	if base.Addr().Is4() {
		raw := [4]byte{}
		for i := range raw {
			raw[i] = rapid.Byte().Draw(t, fmt.Sprintf("%s byte %d", label, i))
		}
		baseRaw := base.Addr().As4()
		copyPrefixBits(raw[:], baseRaw[:], keepBits)
		if flipBit >= 0 {
			mask := byte(1) << (7 - flipBit%8)
			if baseRaw[flipBit/8]&mask != 0 {
				raw[flipBit/8] |= mask
			} else {
				raw[flipBit/8] &^= mask
			}
			flipAddrBit(raw[:], flipBit)
		}
		return netip.AddrFrom4(raw)
	}

	raw := [16]byte{}
	for i := range raw {
		raw[i] = rapid.Byte().Draw(t, fmt.Sprintf("%s byte %d", label, i))
	}
	baseRaw := base.Addr().As16()
	copyPrefixBits(raw[:], baseRaw[:], keepBits)
	if flipBit >= 0 {
		mask := byte(1) << (7 - flipBit%8)
		if baseRaw[flipBit/8]&mask != 0 {
			raw[flipBit/8] |= mask
		} else {
			raw[flipBit/8] &^= mask
		}
		flipAddrBit(raw[:], flipBit)
	}
	return netip.AddrFrom16(raw)
}

// drawDescendantPrefix generates a random bits-length prefix within base.
func drawDescendantPrefix(t *rapid.T, base netip.Prefix, bits int, label string) netip.Prefix {
	addr := drawRelatedAddr(t, base, base.Bits(), -1, label)
	return netip.PrefixFrom(addr, bits).Masked()
}

// drawCousinPrefix generates a same-family prefix that diverges from base at a
// randomly selected bit. For /0, where no disjoint cousin exists, it returns a
// random descendant instead.
func drawCousinPrefix(t *rapid.T, base netip.Prefix, label string) netip.Prefix {
	if base.Bits() == 0 {
		return drawDescendantPrefix(t, base,
			rapid.IntRange(1, prefixMaxBits(base)).Draw(t, label+" bits"), label+" address")
	}
	divergeAt := rapid.IntRange(0, base.Bits()-1).Draw(t, label+" divergence")
	bits := rapid.IntRange(divergeAt+1, prefixMaxBits(base)).Draw(t, label+" bits")
	addr := drawRelatedAddr(t, base, divergeAt, divergeAt, label+" address")
	return netip.PrefixFrom(addr, bits).Masked()
}

// drawStatefulPrefix generates either an independent prefix or one with a
// useful structural relationship to an existing entry.
func drawStatefulPrefix(t *rapid.T, existing []netip.Prefix, label string) netip.Prefix {
	if len(existing) > 0 && rapid.Bool().Draw(t, label+" related") {
		base := existing[rapid.IntRange(0, len(existing)-1).Draw(t, label+" base")]
		maxBits := prefixMaxBits(base)

		switch rapid.IntRange(0, 6).Draw(t, label+" relationship") {
		case 0:
			return base
		case 1:
			bits := rapid.IntRange(0, base.Bits()).Draw(t, label+" ancestor bits")
			return netip.PrefixFrom(base.Addr(), bits).Masked()
		case 2:
			bits := rapid.IntRange(base.Bits(), maxBits).Draw(t, label+" descendant bits")
			return drawDescendantPrefix(t, base, bits, label+" descendant")
		case 3:
			return siblingPrefix(base)
		case 4:
			return drawCousinPrefix(t, base, label+" cousin")
		case 5:
			return drawDescendantPrefix(t, base, maxBits, label+" host")
		default:
			if base.Addr().Is4() {
				return rapid.Custom(genIPv6Prefix).Draw(t, label+" other family IPv6")
			}
			return rapid.Custom(genIPv4Prefix).Draw(t, label+" other family IPv4")
		}
	}

	if rapid.Bool().Draw(t, label+" family") {
		return rapid.Custom(genIPv4Prefix).Draw(t, label+" IPv4")
	}
	return rapid.Custom(genIPv6Prefix).Draw(t, label+" IPv6")
}

// drawOperandPrefixes generates a small operand whose entries may relate to
// existing state or to entries generated earlier in the same operand.
func drawOperandPrefixes(t *rapid.T, existing []netip.Prefix, label string) []netip.Prefix {
	count := rapid.IntRange(0, 4).Draw(t, label+" count")
	prefixes := make([]netip.Prefix, 0, count)
	candidates := append([]netip.Prefix(nil), existing...)
	for i := 0; i < count; i++ {
		p := drawStatefulPrefix(t, candidates, fmt.Sprintf("%s prefix %d", label, i))
		prefixes = append(prefixes, p)
		candidates = append(candidates, p)
	}
	return prefixes
}

type prefixSetOperation uint8

const (
	opAdd prefixSetOperation = iota
	opRemove
	opSubtractPrefix
	opMerge
	opSubtract
	opIntersect
	opFilter
	opCount
)

type prefixSetCommand struct {
	kind            prefixSetOperation
	prefix          netip.Prefix
	operand         *PrefixSet
	operandPrefixes []netip.Prefix
}

func prefixSlicesEqual(a, b []netip.Prefix) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func removeExactPrefix(prefixes []netip.Prefix, remove netip.Prefix) []netip.Prefix {
	remaining := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		if p != remove {
			remaining = append(remaining, p)
		}
	}
	return remaining
}

func buildBuilderRapid(t *rapid.T, prefixes []netip.Prefix, label string) *PrefixSetBuilder {
	b := &PrefixSetBuilder{}
	for _, p := range prefixes {
		if err := b.Add(p); err != nil {
			t.Fatalf("%s Add(%v): %v", label, p, err)
		}
	}
	return b
}

func buildSetRapid(t *rapid.T, prefixes []netip.Prefix, label string) *PrefixSet {
	return buildBuilderRapid(t, prefixes, label).PrefixSet()
}

// drawFreshCommand generates one operation. Set operands are built directly
// from their generated prefixes, terminating nested history generation.
func drawFreshCommand(
	t *rapid.T,
	kind prefixSetOperation,
	existing []netip.Prefix,
	label string,
) prefixSetCommand {
	command := prefixSetCommand{kind: kind}
	switch kind {
	case opAdd, opRemove, opSubtractPrefix:
		command.prefix = drawStatefulPrefix(t, existing, label+" prefix")
	case opMerge, opSubtract, opIntersect, opFilter:
		command.operandPrefixes = drawOperandPrefixes(t, existing, label+" operand")
		command.operand = buildSetRapid(t, command.operandPrefixes, label+" operand")
	}
	return command
}

// apply executes c against b.
func (c prefixSetCommand) apply(t *rapid.T, b *PrefixSetBuilder) {
	switch c.kind {
	case opAdd:
		if err := b.Add(c.prefix); err != nil {
			t.Fatalf("Add(%v): %v", c.prefix, err)
		}
	case opRemove:
		if err := b.Remove(c.prefix); err != nil {
			t.Fatalf("Remove(%v): %v", c.prefix, err)
		}
	case opSubtractPrefix:
		if err := b.SubtractPrefix(c.prefix); err != nil {
			t.Fatalf("SubtractPrefix(%v): %v", c.prefix, err)
		}
	case opMerge:
		b.Merge(c.operand)
	case opSubtract:
		b.Subtract(c.operand)
	case opIntersect:
		b.Intersect(c.operand)
	case opFilter:
		b.Filter(c.operand)
	}
}

// updateOracle returns the expected covered IP space after c. Remove and
// Filter rebuild coverage from exact entries because they do not perform IP
// space subtraction or intersection.
func (c prefixSetCommand) updateOracle(
	t *rapid.T,
	current *netipx.IPSet,
	existing []netip.Prefix,
) *netipx.IPSet {
	switch c.kind {
	case opAdd:
		return mutateIPSet(t, current, func(b *netipx.IPSetBuilder) {
			b.AddPrefix(c.prefix)
		})
	case opRemove:
		return mustIPSet(t, removeExactPrefix(existing, c.prefix))
	case opSubtractPrefix:
		return mutateIPSet(t, current, func(b *netipx.IPSetBuilder) {
			b.RemovePrefix(c.prefix)
		})
	case opMerge:
		operand := mustIPSet(t, c.operandPrefixes)
		return mutateIPSet(t, current, func(b *netipx.IPSetBuilder) {
			b.AddSet(operand)
		})
	case opSubtract:
		operand := mustIPSet(t, c.operandPrefixes)
		return mutateIPSet(t, current, func(b *netipx.IPSetBuilder) {
			b.RemoveSet(operand)
		})
	case opIntersect:
		operand := mustIPSet(t, c.operandPrefixes)
		return mutateIPSet(t, current, func(b *netipx.IPSetBuilder) {
			b.Intersect(operand)
		})
	case opFilter:
		remaining := make([]netip.Prefix, 0, len(existing))
		for _, p := range existing {
			if c.operand.Encompasses(p) {
				remaining = append(remaining, p)
			}
		}
		return mustIPSet(t, remaining)
	default:
		t.Fatalf("unknown operation %d", c.kind)
		return nil
	}
}

func (c prefixSetCommand) String() string {
	switch c.kind {
	case opAdd:
		return "add " + c.prefix.String()
	case opRemove:
		return "remove " + c.prefix.String()
	case opSubtractPrefix:
		return "subtract-prefix " + c.prefix.String()
	case opMerge:
		return fmt.Sprintf("merge %v", c.operandPrefixes)
	case opSubtract:
		return fmt.Sprintf("subtract %v", c.operandPrefixes)
	case opIntersect:
		return fmt.Sprintf("intersect %v", c.operandPrefixes)
	case opFilter:
		return fmt.Sprintf("filter %v", c.operandPrefixes)
	default:
		return fmt.Sprintf("operation %d", c.kind)
	}
}

type savedPrefixSet struct {
	set      *PrefixSet
	prefixes []netip.Prefix
	compact  []netip.Prefix
	size     int
}

// savePrefixSet records the observable state of an immutable snapshot.
func savePrefixSet(set *PrefixSet) savedPrefixSet {
	return savedPrefixSet{
		set:      set,
		prefixes: set.Prefixes(),
		compact:  set.PrefixesCompact(),
		size:     set.Size(),
	}
}

// assertSnapshotsUnchanged verifies that saved immutable sets retain their
// structure and behavior as their source builders continue to mutate.
func assertSnapshotsUnchanged(t *rapid.T, snapshots []savedPrefixSet, label string) {
	for i, snapshot := range snapshots {
		assertTreeInvariants(t, &snapshot.set.tree4, 32, fmt.Sprintf("%s snapshot %d IPv4", label, i))
		assertTreeInvariants(t, &snapshot.set.tree6, 128, fmt.Sprintf("%s snapshot %d IPv6", label, i))
		if got := snapshot.set.Prefixes(); !prefixSlicesEqual(got, snapshot.prefixes) {
			t.Fatalf("%s snapshot %d changed:\nwant: %v\ngot: %v", label, i, snapshot.prefixes, got)
		}
		if got := snapshot.set.PrefixesCompact(); !prefixSlicesEqual(got, snapshot.compact) {
			t.Fatalf("%s compact snapshot %d changed:\nwant: %v\ngot: %v",
				label, i, snapshot.compact, got)
		}
		if got := snapshot.set.Size(); got != snapshot.size {
			t.Fatalf("%s snapshot %d size changed: want %d, got %d", label, i, snapshot.size, got)
		}
		fresh := buildSetRapid(t, snapshot.prefixes, fmt.Sprintf("%s snapshot %d rebuild", label, i))
		assertPrefixSetsEquivalent(t, snapshot.set, fresh, snapshot.prefixes,
			fmt.Sprintf("%s snapshot %d", label, i))
	}
}

// drawBuilderHistory constructs a builder through both simple mutations and
// set operations. Its operands are built directly, which bounds generated
// history nesting at one level.
func drawBuilderHistory(
	t *rapid.T,
	seeds []netip.Prefix,
	label string,
	maxSteps int,
) (*PrefixSetBuilder, []string) {
	b := &PrefixSetBuilder{}
	oracle := mustIPSet(t, nil)
	history := make([]string, 0, len(seeds)+maxSteps)
	snapshots := make([]savedPrefixSet, 0, len(seeds)+maxSteps)

	apply := func(command prefixSetCommand) {
		before := assertBuilderInvariants(t, b)
		existing := before.Prefixes()
		snapshots = append(snapshots, savePrefixSet(before))
		oracle = command.updateOracle(t, oracle, existing)
		command.apply(t, b)
		history = append(history, command.String())
		assertBuilderMatchesOracle(t, b, oracle, label)
		assertSnapshotsUnchanged(t, snapshots, label)
	}

	for _, p := range seeds {
		apply(prefixSetCommand{kind: opAdd, prefix: p})
	}

	// Explicitly exercise branch contraction before continuing with arbitrary
	// operations. This keeps removal regressions likely without making removal
	// the only mutation history under test.
	removals := rapid.IntRange(0, len(b.PrefixSet().Prefixes())).Draw(t, label+" initial removals")
	for i := 0; i < removals; i++ {
		existing := b.PrefixSet().Prefixes()
		remove := existing[rapid.IntRange(0, len(existing)-1).Draw(t,
			fmt.Sprintf("%s initial removal %d", label, i))]
		apply(prefixSetCommand{kind: opRemove, prefix: remove})
	}

	steps := rapid.IntRange(0, maxSteps).Draw(t, label+" history length")
	for step := 0; step < steps; step++ {
		existing := b.PrefixSet().Prefixes()
		kind := prefixSetOperation(rapid.IntRange(0, int(opCount)-1).Draw(t,
			fmt.Sprintf("%s history operation %d", label, step)))
		apply(drawFreshCommand(t, kind, existing, fmt.Sprintf("%s history %d", label, step)))
	}

	return b, history
}

// assertTreeInvariants validates root identity, compressed-edge relationships,
// branching shape, entry values, and the absence of cycles or shared nodes. It
// returns the number of entries encountered.
func assertTreeInvariants[B keybits[B]](t *rapid.T, root *tree[bool, B], maxBits uint8, name string) int {
	if !root.key.IsZero() || root.key.offset != 0 || !root.key.content.IsZero() {
		t.Fatalf("%s root has invalid key %v", name, root.key)
	}

	seen := make(map[*tree[bool, B]]struct{})
	entries := 0
	var visit func(*tree[bool, B], *tree[bool, B], bit)
	visit = func(n, parent *tree[bool, B], direction bit) {
		if _, ok := seen[n]; ok {
			t.Fatalf("%s tree contains a cycle or shared node at %v", name, n.key)
		}
		seen[n] = struct{}{}

		if n.key.len > maxBits || n.key.offset > n.key.len {
			t.Fatalf("%s node has invalid key bounds %v", name, n.key)
		}
		if n.key.content != n.key.content.BitsClearedFrom(n.key.len) {
			t.Fatalf("%s node key has bits set beyond its prefix length: %v", name, n.key)
		}

		if parent != nil {
			if n.key.len <= parent.key.len || n.key.offset != parent.key.len {
				t.Fatalf("%s child %v has invalid relationship to parent %v", name, n.key, parent.key)
			}
			if !parent.key.IsPrefixOf(n.key) || n.key.Bit(parent.key.len) != direction {
				t.Fatalf("%s child %v is on the wrong branch of parent %v", name, n.key, parent.key)
			}
			if n.isEmpty() {
				t.Fatalf("%s contains an empty non-root node at %v", name, n.key)
			}
			if !n.hasEntry && (n.left == nil || n.right == nil) {
				t.Fatalf("%s contains an uncompressed node at %v", name, n.key)
			}
		}

		if n.hasEntry {
			entries++
			if !n.value {
				t.Fatalf("%s entry at %v has a false value", name, n.key)
			}
		}
		if n.left != nil {
			visit(n.left, n, bitL)
		}
		if n.right != nil {
			visit(n.right, n, bitR)
		}
	}

	visit(root, nil, bitL)
	return entries
}

// assertBuilderInvariants validates both address-family trees and returns an
// immutable snapshot whose stored sizes agree with their entry counts.
func assertBuilderInvariants(t *rapid.T, b *PrefixSetBuilder) *PrefixSet {
	entries4 := assertTreeInvariants(t, &b.tree4, 32, "IPv4")
	entries6 := assertTreeInvariants(t, &b.tree6, 128, "IPv6")
	snapshot := b.PrefixSet()
	if snapshot.size4 != entries4 || snapshot.size6 != entries6 {
		t.Fatalf("stored sizes are (%d, %d), want (%d, %d)",
			snapshot.size4, snapshot.size6, entries4, entries6)
	}
	return snapshot
}

// assertBuilderMatchesOracle validates b's structure and compares its covered
// IP space with the independent netipx oracle.
func assertBuilderMatchesOracle(
	t *rapid.T,
	b *PrefixSetBuilder,
	oracle *netipx.IPSet,
	label string,
) *PrefixSet {
	actual := assertBuilderInvariants(t, b)
	actualIPSet := mustIPSet(t, actual.Prefixes())
	if !actualIPSet.Equal(oracle) {
		t.Fatalf("%s coverage mismatch:\nwant: %v\ngot: %v",
			label, oracle.Prefixes(), actualIPSet.Prefixes())
	}
	return actual
}

// assertPrefixSetsEquivalent compares enumeration and every prefix-query API
// for two sets that should differ only in their construction histories.
func assertPrefixSetsEquivalent(
	t *rapid.T,
	a, b *PrefixSet,
	probes []netip.Prefix,
	label string,
) {
	if got, want := a.Prefixes(), b.Prefixes(); !prefixSlicesEqual(got, want) {
		t.Fatalf("%s Prefixes differ:\nwant: %v\ngot: %v", label, want, got)
	}
	if got, want := a.PrefixesCompact(), b.PrefixesCompact(); !prefixSlicesEqual(got, want) {
		t.Fatalf("%s PrefixesCompact differ:\nwant: %v\ngot: %v", label, want, got)
	}
	if a.Size() != b.Size() {
		t.Fatalf("%s sizes differ: %d != %d", label, a.Size(), b.Size())
	}

	for _, p := range probes {
		if a.Contains(p) != b.Contains(p) {
			t.Fatalf("%s Contains(%v) differs", label, p)
		}
		if a.Encompasses(p) != b.Encompasses(p) {
			t.Fatalf("%s Encompasses(%v) differs", label, p)
		}
		if a.OverlapsPrefix(p) != b.OverlapsPrefix(p) {
			t.Fatalf("%s OverlapsPrefix(%v) differs", label, p)
		}
		aRoot, aOK := a.RootOf(p)
		bRoot, bOK := b.RootOf(p)
		if aRoot != bRoot || aOK != bOK {
			t.Fatalf("%s RootOf(%v) differs: (%v, %v) != (%v, %v)",
				label, p, aRoot, aOK, bRoot, bOK)
		}
		aParent, aOK := a.ParentOf(p)
		bParent, bOK := b.ParentOf(p)
		if aParent != bParent || aOK != bOK {
			t.Fatalf("%s ParentOf(%v) differs: (%v, %v) != (%v, %v)",
				label, p, aParent, aOK, bParent, bOK)
		}
		if got, want := a.Subnets(p).Prefixes(), b.Subnets(p).Prefixes(); !prefixSlicesEqual(got, want) {
			t.Fatalf("%s Subnets(%v) differ:\nwant: %v\ngot: %v", label, p, want, got)
		}
		if got, want := a.Supernets(p).Prefixes(), b.Supernets(p).Prefixes(); !prefixSlicesEqual(got, want) {
			t.Fatalf("%s Supernets(%v) differ:\nwant: %v\ngot: %v", label, p, want, got)
		}
	}
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

// TestPrefixSetOperandShapeProp checks that set operations depend only on the
// observable contents of their operands, not on the mutation histories that
// produced their internal trees.
func TestPrefixSetOperandShapeProp(t *testing.T) {
	operations := []struct {
		name string
		kind prefixSetOperation
	}{
		{"Merge", opMerge},
		{"Subtract", opSubtract},
		{"Intersect", opIntersect},
		{"Filter", opFilter},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			rapid.Check(t, func(rt *rapid.T) {
				seeds := drawOperandPrefixes(rt, nil, "operand seeds")
				historyBuilder, history := drawBuilderHistory(rt, seeds, "operand", 6)
				historySet := assertBuilderInvariants(rt, historyBuilder)
				contents := historySet.Prefixes()
				freshSet := buildSetRapid(rt, contents, "fresh operand")

				target := drawOperandPrefixes(rt, append(seeds, contents...), "target")
				probes := append(append([]netip.Prefix(nil), seeds...), contents...)
				probes = append(probes, target...)
				probes = append(probes, drawOperandPrefixes(rt, probes, "query probes")...)
				assertPrefixSetsEquivalent(rt, historySet, freshSet, probes, "operand")

				historyResult := buildBuilderRapid(rt, target, "history target")
				freshResult := buildBuilderRapid(rt, target, "fresh target")
				command := prefixSetCommand{
					kind:            op.kind,
					operand:         historySet,
					operandPrefixes: contents,
				}
				command.apply(rt, historyResult)
				command.operand = freshSet
				command.apply(rt, freshResult)

				historyResultSet := assertBuilderInvariants(rt, historyResult)
				freshResultSet := assertBuilderInvariants(rt, freshResult)
				assertPrefixSetsEquivalent(rt, historyResultSet, freshResultSet, probes,
					fmt.Sprintf("%s after history %v", op.name, history))
			})
		})
	}
}

func drawHistoryOperand(
	t *rapid.T,
	existing []netip.Prefix,
	label string,
) (*PrefixSet, []netip.Prefix, []string) {
	seeds := drawOperandPrefixes(t, existing, label+" seeds")
	b, history := drawBuilderHistory(t, seeds, label, 4)
	set := assertBuilderInvariants(t, b)
	return set, set.Prefixes(), history
}

func drawStatefulCommand(
	t *rapid.T,
	kind prefixSetOperation,
	existing []netip.Prefix,
	label string,
) (prefixSetCommand, []string) {
	if kind < opMerge {
		return drawFreshCommand(t, kind, existing, label), nil
	}
	operand, prefixes, history := drawHistoryOperand(t, existing, label+" operand")
	return prefixSetCommand{
		kind:            kind,
		operand:         operand,
		operandPrefixes: prefixes,
	}, history
}

func statefulProbes(
	t *rapid.T,
	existing []netip.Prefix,
	command prefixSetCommand,
	label string,
) []netip.Prefix {
	candidates := append([]netip.Prefix(nil), existing...)
	candidates = append(candidates, command.operandPrefixes...)
	probes := make([]netip.Prefix, 0, 5)
	if command.prefix.IsValid() {
		probes = append(probes, command.prefix)
	}
	probes = append(probes, drawOperandPrefixes(t, candidates, label)...)
	return probes
}

// TestPrefixSetBuilderStatefulProp applies mixed operation sequences while
// checking structure, coverage, snapshot immutability, and equivalence to a
// freshly rebuilt tree after every transition.
func TestPrefixSetBuilderStatefulProp(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var b PrefixSetBuilder
		oracle := mustIPSet(rt, nil)
		operations := make([]string, 0)
		snapshots := make([]savedPrefixSet, 0)
		steps := rapid.IntRange(int(opCount), 30).Draw(rt, "operation count")

		for step := 0; step < steps; step++ {
			before := assertBuilderInvariants(rt, &b)
			existing := before.Prefixes()
			snapshots = append(snapshots, savePrefixSet(before))

			kind := prefixSetOperation(step)
			if kind >= opCount {
				kind = prefixSetOperation(rapid.IntRange(0, int(opCount)-1).Draw(rt,
					fmt.Sprintf("operation %d", step)))
			}
			command, operandHistory := drawStatefulCommand(rt, kind, existing,
				fmt.Sprintf("operation %d", step))

			// Apply the same transition to the history-built tree and to a fresh
			// reconstruction of its observable entries. Any difference reveals
			// behavior that depends on hidden tree shape.
			fresh := buildBuilderRapid(rt, existing, fmt.Sprintf("fresh state %d", step))
			oracle = command.updateOracle(rt, oracle, existing)
			command.apply(rt, &b)
			command.apply(rt, fresh)

			description := command.String()
			if len(operandHistory) != 0 {
				description += fmt.Sprintf(" via %v", operandHistory)
			}
			operations = append(operations, description)
			rt.Logf("operations: %v", operations)

			actual := assertBuilderMatchesOracle(rt, &b, oracle,
				fmt.Sprintf("stateful step %d", step))
			freshSet := assertBuilderInvariants(rt, fresh)
			assertPrefixSetsEquivalent(rt, actual, freshSet,
				statefulProbes(rt, existing, command, fmt.Sprintf("step %d probes", step)),
				fmt.Sprintf("stateful step %d after %v", step, operations))
			assertSnapshotsUnchanged(rt, snapshots, "stateful")
		}
	})
}
