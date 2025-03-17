package netipds

// bit is used as a selector for a node's children.
//
// bitL refers to the left child, and bitR to the right.
type bit = bool

const (
	bitL = false
	bitR = true
)

var eachBit = [2]bit{bitL, bitR}
