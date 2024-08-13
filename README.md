# netipds
[![Go Reference](https://pkg.go.dev/badge/github.com/aromatt/netipds)](https://pkg.go.dev/github.com/aromatt/netipds)
[![Go Report Card](https://goreportcard.com/badge/github.com/aromatt/netipds)](https://goreportcard.com/report/github.com/aromatt/netipds)
[![codecov](https://codecov.io/gh/aromatt/netipds/graph/badge.svg?token=WJ1JHSM05F)](https://codecov.io/gh/aromatt/netipds)

This package builds on the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, adding two new collection types:
* `PrefixMap[T]`, an immutable, tree-based map with `netip.Prefix` keys
* `PrefixSet`, an immutable set type for `netip.Prefix`, offering better performance
  and a more comprehensive feature set than
  [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet)

Both are backed by a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree).

## Project Goals
* Provide efficient, thread-safe, immutable collection types for IP networks
* Integrate well with the `netip` and `netipx` packages
* Support use cases that are not covered by other libraries

## Usage
Usage is similar to that of [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet):
to construct a `PrefixMap` or `PrefixSet`, use the respective builder type.

### Example
```go
// Build a PrefixMap
psb := netipds.PrefixMapBuilder[string]
pmb.Set(netip.MustParsePrefix("1.2.0.0/16"), "hello")
pmb.Set(netip.MustParsePrefix("1.2.3.0/24"), "world")
pm := pmb.PrefixMap()

// (Prepare some Prefixes for queries)
p8 := netip.MustParsePrefix("1.0.0.0/8")
p16 := netip.MustParsePrefix("1.2.0.0/16")
p24 := netip.MustParsePrefix("1.2.3.0/24")
p32 := netip.MustParsePrefix("1.2.3.4/32")

// Fetch an exact entry from the PrefixMap.
val, ok := pm.Get(p16) // => ("hello", true)

// Ask if the PrefixMap contains an exact entry.
ok = pm.Contains(p32) // => false

// Ask if a Prefix has any ancestor in the PrefixMap.
ok = pm.Encompasses(p32) // => true

// Fetch a Prefix's nearest ancestor.
prefix, val, ok := pm.ParentOf(p32) // => (1.2.3.0/24, "world", true)

// Fetch all of a Prefix's ancestors, and convert the result to a map[Prefix]string.
m := pm.AncestorsOf(p32).ToMap() // map[1.2.0.0/16:"hello" 1.2.3.0/24:"world"]

// Fetch all of a Prefix's descendants, and convert the result to a map[Prefix]string.
m = pm.DescendantsOf(p8).ToMap() // map[1.2.0.0/16:"hello" 1.2.3.0/24:"world"]
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

## Related packages

### https://github.com/kentik/patricia

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost. By contrast, netipds aims to
provide immutable (and thus GC-friendly) collection types that integrate well with
the netip family and offer a comprehensive API.
