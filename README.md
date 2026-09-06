# netipds
[![Go Reference](https://pkg.go.dev/badge/github.com/aromatt/netipds)](https://pkg.go.dev/github.com/aromatt/netipds)
[![Go](https://github.com/aromatt/netipds/actions/workflows/go.yml/badge.svg)](https://github.com/aromatt/netipds/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/aromatt/netipds/graph/badge.svg?token=WJ1JHSM05F)](https://codecov.io/gh/aromatt/netipds)

This package builds on the
[netip](https://pkg.go.dev/net/netip) / [netipx](https://pkg.go.dev/go4.org/netipx)
family by adding two immutable, trie-based collection types for IP prefixes (CIDRs):
* `PrefixMap[T]` - a map from `netip.Prefix` to `T` supporting CIDR-based retrieval
  (longest-match, subnets, supernets, etc.)
* `PrefixSet` - a set of `netip.Prefix` values supporting [semantic combination](#combining-sets-and-maps)

### Goals
* **Efficiency.** This package aims to provide fast, immutable collection types for
  IP networks. According to the benchmarks at
  [iprbench](https://github.com/gaissmai/iprbench), it is one of the fastest and most
  memory-efficient packages among its peers.
* **Integration with `net/netip`.** This package is built on the shoulders of
  `net/netip`, leveraging its types and lessons both under the hood and at
  interfaces. See this excellent
  [post](https://tailscale.com/blog/netaddr-new-ip-type-for-go) by Tailscale about
  the history and benefits of `net/netip`.
* **Completeness.** Most open-source CIDR collection libraries lack several of the
  operations provided by `netipds`.

### Non-Goals
* **Mutability.** For use cases requiring continuous mutability, try
  [kentik/patricia](https://github.com/kentik/patricia) or
  [gaissmai/bart](https://github.com/gaissmai/bart).
* **Persistence.** This package is for data sets that fit in memory.
* **Other key types.** The collections in this package support exactly one key type:
  `netip.Prefix`.

## Usage
Like [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet), `netipds` uses a
builder pattern to construct immutable PrefixMaps and PrefixSets.

Basic Example
```go
// Build a PrefixMap
builder := PrefixMapBuilder[string]{}
builder.Set(netip.MustParsePrefix("1.2.0.0/16"), "hello")

// This returns an immutable snapshot of the
// builder's state. The builder remains usable.
pm := builder.PrefixMap()

// Fetch an exact entry from the PrefixMap.
val, ok := pm.Get(netip.MustParsePrefix("1.2.0.0/16"))    // => ("hello", true)
```

<details>
<summary>Extended Example</summary>
<br>

```go
// Make our examples more readable
px := netip.MustParsePrefix

// Build a PrefixMap
builder := PrefixMapBuilder[string]{}
builder.Set(px("1.2.0.0/16"), "hello")
builder.Set(px("1.2.3.0/24"), "world")

// This returns an immutable snapshot of the
// builder's state. The builder remains usable.
pm := builder.PrefixMap()

// Fetch an exact entry from the PrefixMap.
val, ok := pm.Get(px("1.2.0.0/16"))              // => ("hello", true)

// Ask if the PrefixMap contains an exact
// entry.
ok = pm.Contains(px("1.2.3.4/32"))               // => false

// Ask if a Prefix has any ancestor in the
// PrefixMap.
ok = pm.Encompasses(px("1.2.3.4/32"))            // => true

// Fetch a Prefix's nearest ancestor.
p, val, ok := pm.ParentOf(px("1.2.3.4/32"))      // => (1.2.3.0/24, "world", true)

// Fetch all of a Prefix's ancestors, and
// convert the result to a map[Prefix]string.
m := pm.AncestorsOf(px("1.2.3.4/32")).ToMap()    // => map[1.2.0.0/16:"hello"
                                                 //        1.2.3.0/24:"world"]

// Fetch all of a Prefix's descendants, and
// convert the result to a map[Prefix]string.
m = pm.DescendantsOf(px("1.0.0.0/8")).ToMap()    // => map[1.2.0.0/16:"hello"
                                                 //        1.2.3.0/24:"world"]
```
</details>

See [docs](https://pkg.go.dev/github.com/aromatt/netipds) for more details.

## API Tour
### Membership Queries
Both PrefixMaps and PrefixSets support the following queries:

* [Contains](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.Contains) - Ask if the collection contains an exact prefix.
* [Encompasses](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.Encompasses) - Ask if the collection contains any supernets of a prefix.
* [OverlapsPrefix](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.OverlapsPrefix) - Ask if the collection has any overlap with a prefix.
* [ParentOf](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.ParentOf) - Fetch a prefix's smallest supernet in the collection (longest-prefix match).
* [RootOf](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.RootOf) - Fetch a prefix's largest supernet in the collection (shortest-prefix match).
* [AncestorsOf](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.AncestorsOf) - Fetch all of a prefix's supernets in the collection.
* [DescendantsOf](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMap.DescendantsOf) - Fetch all of a prefix's subnets in the collection.

### Combining Sets and Maps
During the build stage, `netipds` collections can be combined in the following ways:
* [Filter](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMapBuilder.Filter) - Filters a collection, removing all prefixes not encompassed by the provided set.
* [Merge](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Merge) - Merges two sets. The result is the union of the two sets.
* [Intersect](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Intersect) - Performs a hierarchical intersection of two sets. The result includes every prefix that either (1) exists in both sets or (2) exists in one set and has an ancestor in the other.
* [Subtract](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Subtract) - Subtracts one set's IP space from the other, adding new child prefixes if necessary to fill in gaps around subtracted IP space.

## Errors
Not every possible value of `netip.Prefix` is a valid IP prefix. For example,
`netip.Prefix{}` is invalid. CIDR collection libraries such as `netipds` handle
invalid prefixes in different ways. `netipds` takes the following approach:

_If an invalid prefix is provided to a PrefixMapBuilder or PrefixSetBuilder,
then an error is returned immediately and the builder remains valid._

Rationale: returning errors at the mutation call site is familiar and unopinionated,
allowing callers to decide between failing immediately, collecting errors for later,
or explicitly ignoring them.

Example patterns:

### 1. Fail fast
If you want to build a collection from unvetted prefixes and let `netipds` tell you
about the invalid ones right away, you can do that:

```go
for _, p := range prefixes {
    if err := builder.Add(p); err != nil {
        return err
    }
}
```

### 2. Batch errors
This pattern is used by [netipx](https://pkg.go.dev/go4.org/netipx), which
accumulates errors during the build phase, then returns them as a batch.

```go
var errs []error
for _, p := range prefixes {
    if err := builder.Add(p); err != nil {
        errs = append(errs, err)
    }
}
// later...
return errors.Join(errs...)
```

`netipds` tries to follow the idioms of `netipx` in general, but forced
error-batching adds extra internal machinery, is not what all users want, and is
easily implemented on top of the `netipds` API.

### 3. Silently skip invalid prefixes
This pattern is used by [bart](https://pkg.go.dev/github.com/gaissmai/bart), which
does not return errors at all for insertions (invalid prefixes result in no-ops).

```go
for _, p := range prefixes {
    _ = builder.Add(p)
}
```

Use this pattern when invalid prefixes can safely be ignored.

## Note about Value-Copying in PrefixMap
When generating an immutable PrefixMap from a PrefixMapBuilder (using
[PrefixMapBuilder.PrefixMap](https://pkg.go.dev/github.com/aromatt/netipds#PrefixMapBuilder.PrefixMap)),
the builder's values are copied by assignment, so please be careful if you are
storing pointers (https://github.com/aromatt/netipds/issues/27 aims to improve this).

The same warning applies to any PrefixMap method that returns a new PrefixMap.

## Tests
In addition to 100% line coverage in unit tests, `netipds` includes [property-based
tests](prefixset_property_test.go).

These tests generate large sets of random inputs and test for properties such as
commutativity and exact parity against reference implementations.

## Related Packages

### [gaissmai/bart](https://github.com/gaissmai/bart)

This package uses a variant of Donald Knuth's ART algorithm. It provides mutability
while optimizing for lookup time and memory usage. Its API also provides several
useful methods.

By contrast, `netipds` uses a traditional trie implementation, provides immutable
types using a builder pattern, and offers additional features.

### [tailscale/art](https://github.com/tailscale/art)

An inspiration for bart, this package is also based on Knuth's ART algorithm. It
provides good lookup performance and a barebones API.

It is not actively maintained; in fact, Tailscale uses bart in some of its
open-source systems.

### [kentik/patricia](https://github.com/kentik/patricia)

This package focuses on providing mutability while minimizing garbage collection
cost.

By contrast, `netipds` aims to provide immutable collections with a more
comprehensive API.

## Performance
The benchmark suite at [gaissmai/iprbench](https://github.com/gaissmai/iprbench)
compares several "IP routing table implementations," including `netipds`.

As with any benchmark, these results do not necessarily reflect real-world
performance. In the currently published results:

- **Lookup time:** `netipds` is the only implementation other than the benchmark
  author's (`bart`) with a longest-prefix-match geomean below 30 nanoseconds.
- **Memory usage:** `netipds` has the lowest memory usage other than `bart.Lite`, the
  author's memory-optimized variant.
- **Update time:** `netipds` uses builders to construct immutable collections, so it
  is not optimized for update time; however it still beats most of the
  implementations in this benchmark.

## Pre-1.0 Breaking Changes
The following versions have breaking API changes:
* v0.1.9: Removed `Strict` methods, removed `PrefixMapBuilder.Get`, moved `String`
  methods behind build tag and renamed them, removed `Lazy` mode
