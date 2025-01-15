# netipds
[![Go Reference](https://pkg.go.dev/badge/github.com/aromatt/netipds)](https://pkg.go.dev/github.com/aromatt/netipds)
[![Go Report Card](https://goreportcard.com/badge/github.com/aromatt/netipds)](https://goreportcard.com/report/github.com/aromatt/netipds)
[![codecov](https://codecov.io/gh/aromatt/netipds/graph/badge.svg?token=WJ1JHSM05F)](https://codecov.io/gh/aromatt/netipds)

This package builds on the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family by adding two immutable, tree-based collection types for [netip.Prefix](https://pkg.go.dev/net/netip#Prefix):
* `PrefixMap[T]` - for associating data with IPs and prefixes and fetching that data with network hierarchy awareness
* `PrefixSet` - for storing sets of prefixes and combining those sets in useful ways (unions, intersections, etc)

Both are backed by a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree),
which enables a rich set of efficient queries about prefix containment, hierarchy,
and overlap.

### Goals
* **Efficiency.** This package aims to provide fast, immutable collection types for IP networks.
* **Integration with `net/netip`.** This package is built on the shoulders of `net/netip`, leveraging its types and lessons both under the hood and at interfaces. See this excellent [post](https://tailscale.com/blog/netaddr-new-ip-type-for-go) by Tailscale about the history and benefits of `net/netip`.
* **Completeness.** Most other IP radix tree libraries lack several of the queries provided by `netipds`.

### Non-Goals
* **Mutability.** For use cases requiring continuous mutability, try [kentik/patricia](https://github.com/kentik/patricia) or [gaissmai/bart](https://github.com/gaissmai/bart).
* **Persistence.** This package is for data sets that fit in memory.
* **Other key types.** The collections in this package support exactly one key type: `netip.Prefix`.

## Usage
Usage is similar to that of [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet):
to construct a `PrefixMap` or `PrefixSet`, use the respective builder type.

### Example
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
val, ok := pm.Get(px("1.0.0.0/16"))              // => ("hello", true)

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

### Set Operations with PrefixSet
`PrefixSet` offers set-specific functionality beyond what can be done with
`PrefixMap`.

In particular, during the building stage, you can combine sets in the following ways:

|Operation|Method|Result|
|---|---|---|
|**Union**|[PrefixSetBuilder.Merge](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Merge)|Every prefix found in either set.|
|**Intersection**|[PrefixSetBuilder.Intersect](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Intersect)|Every prefix that either (1) exists in both sets or (2) exists in one set and has an ancestor in the other.|
|**Difference**|[PrefixSetBuilder.Subtract](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Subtract)|The difference between the two sets. When a child is subtracted from a parent, the child itself is removed, and new elements are added to fill in remaining space.|

## Related Packages

### [kentik/patricia](https://github.com/kentik/patricia)

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost.

By contrast, `netipds` aims to provide immutable collection types that integrate well
with the netip family and offer a comprehensive API.

### [gaissmai/bart](https://github.com/gaissmai/bart)

This package uses a different trie implementation based on the ART algorithm (Knuth).
It provides mutability while optimizing for lookup time and memory usage. Its API
also provides useful methods such as `Subnets`, `Supernets`, `Union`, and iterators.

By contrast, `netipds` uses a traditional trie implementation, provides immutable
types, and offers additional set operations.

### Additional packages
[gaissmai/iprbench](https://github.com/gaissmai/iprbench) is a suite of benchmarks
comparing the performance of several similar libaries.
