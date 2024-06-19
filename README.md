# netipds
[![Go Reference](https://pkg.go.dev/badge/github.com/aromatt/netipds)](https://pkg.go.dev/github.com/aromatt/netipds)
[![Go Report Card](https://goreportcard.com/badge/github.com/aromatt/netipds)](https://goreportcard.com/report/github.com/aromatt/netipds)
[![codecov](https://codecov.io/gh/aromatt/netipds/graph/badge.svg?token=WJ1JHSM05F)](https://codecov.io/gh/aromatt/netipds)

This package builds on the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, adding two new collection types:
* `PrefixMap[T]`, an immutable, tree-based map with `netip.Prefix` keys
* `PrefixSet`, an immutable set type for `netip.Prefix`, offering better performance
  and a more comprehensive API than
  [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet)

Both are backed by a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree)
with path compression.

## Project Goals
* Provide efficient, thread-safe, immutable collection types for IP networks
* Integrate well with the `netip` and `netipx` packages
* Support some use cases that are unsupported by other libraries

## Usage
Usage is similar to that of `IPSet`: to construct a `PrefixMap` or `PrefixSet`, use
the respective builder type.

## Related packages

### https://github.com/kentik/patricia

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost. By contrast, netipds aims to
provide immutable (and thus GC-friendly) collection types that integrate well with
the netip family and offer a comprehensive API.
