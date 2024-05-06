# netipmap
This project adds to the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, providing an immutable associative data structure for IPs and prefixes.

It accepts [netip](https://pkg.go.dev/net/netip) Addrs and Prefixes for keys, and
uses generics for values.

It is implemented as a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree)
with path compression.

## Project Goals
* Provide an efficient, thread-safe, immutable map type for IPs and prefixes
* Integrate well with the netip and netipx packages
* Support use cases that are difficult or impossible with other popular trie packages

## Usage
Usage is similar to that of IPSet: use the IPMapBuilder type to construct an
IPMap.

# Related packages

## https://github.com/kentik/patricia

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost. By contrast, netipmap aims to
provide an immutable (and thus GC-friendly) map type that integrates well with the
netip family.
