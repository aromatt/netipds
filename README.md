# netip-map
This project adds to the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, providing an associative data structure for IP networks.

It accepts [netip.Prefixes](https://pkg.go.dev/net/netip#Prefix) for keys, and uses
generics for values.

It is implemented as a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree).

# Overview

# Related packages
* https://github.com/kentik/patricia

This package

