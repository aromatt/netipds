package netipds

import (
	"strings"
	"testing"
)

func assertMalformedTreePanic(t *testing.T) {
	t.Helper()
	got := recover()
	message, ok := got.(string)
	if !ok || !strings.HasPrefix(message, "netipds: malformed tree:") {
		t.Fatalf("got panic %q, want a malformed-tree panic", got)
	}
}

// TestSubtractTreeImplRejectsMalformedDescent checks that subtraction rejects
// operands outside the target subtree.
func TestSubtractTreeImplRejectsMalformedDescent(t *testing.T) {
	tests := map[string]struct {
		target, other string
	}{
		"other is an ancestor of target": {"0.0.0.0/2", "0.0.0.0/1"},
		"target and other are disjoint":  {"0.0.0.0/1", "128.0.0.0/1"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer assertMalformedTreePanic(t)
			target := newTree[int](key4FromPrefix(pfx(tt.target))).setValue(1)
			other := newTree[int](key4FromPrefix(pfx(tt.other))).setValue(2)
			target.subtractTreeImpl(other, 0, false)
		})
	}
}

// TestInsertHoleRejectsMalformedDescent checks that insertHole rejects holes
// outside the target subtree.
func TestInsertHoleRejectsMalformedDescent(t *testing.T) {
	tests := map[string]struct {
		target, hole string
	}{
		"hole is an ancestor of target": {"0.0.0.0/2", "0.0.0.0/1"},
		"hole is disjoint from target":  {"0.0.0.0/1", "128.0.0.0/1"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer assertMalformedTreePanic(t)
			target := newTree[int](key4FromPrefix(pfx(tt.target))).setValue(1)
			target.insertHole(key4FromPrefix(pfx(tt.hole)), 2)
		})
	}
}
