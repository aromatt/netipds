# netipds
Additional collection types for netip.

# What
This project builds on the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, adding two new collection types:
* PrefixMap, an immutable map for netip.Prefixes
* PrefixSet, an immutable set type for netip.Prefixes (offering better performance
  and a more comprehensive API than netipx.IPSet)

Both accept netip.Prefixes for keys. PrefixMap uses generics for values.

Both are backed by a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree)
with path compression.

## Project Goals
* Provide efficient, thread-safe, immutable collection types for Prefixes
* Integrate well with the netip and netipx packages
* Support use cases that are difficult or impossible with other popular trie
  libraries

## Usage
Usage is similar to that of IPSet: to construct a PrefixMap or PrefixSet, use the
respective builder type.

# Related packages

## https://github.com/kentik/patricia

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost. By contrast, netipds aims to
provide immutable (and thus GC-friendly) collection types that integrate well with
the netip family and offer a comprehensive API.
