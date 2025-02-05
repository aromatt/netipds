package netipds

// bit is used as a selector for a node's children.
//
// bitL refers to the left child, and bitR to the right.
type bit = uint8

const (
	bitL = 0
	bitR = 1
)

var eachBit = [2]bit{bitL, bitR}
