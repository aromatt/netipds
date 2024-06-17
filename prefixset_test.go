package netipds

import (
	"net/netip"
	"testing"
)

func TestPrefixSetAddContains(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), false},
		{pfxs("::0/127"), pfx("::0/128"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), false},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Contains(tt.get); got != tt.want {
			t.Errorf("pm.Contains(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixSetAddEncompasses(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), false},
		{pfxs("::0/127"), pfx("::0/128"), true},
		// The set covers the input prefix but does not encompass it.
		{pfxs("::0/128", "::1/128"), pfx("::0/127"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), true},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Encompasses(tt.get); got != tt.want {
			t.Errorf("pm.Encompasses(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func TestPrefixSetOverlapsPrefix(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want bool
	}{
		{pfxs(), pfx("::0/128"), false},
		{pfxs("::0/128"), pfx("::0/128"), true},
		{pfxs("::0/128"), pfx("::1/128"), false},
		{pfxs("::0/128"), pfx("::0/127"), true},
		{pfxs("::0/127"), pfx("::0/128"), true},
		{pfxs("::0/128", "::1/128"), pfx("::2/128"), false},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.0/24"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.3.4/32"), true},
		{pfxs("1.2.3.0/24"), pfx("1.2.0.0/16"), true},

		// Make sure value-less nodes don't count. This map contains
		// the shared prefix ::0/126.
		{pfxs("::0/128", "::2/128"), pfx("::3/128"), false},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.Overlaps(tt.get); got != tt.want {
			t.Errorf("pm.OverlapsPrefix(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func checkPrefixSlice(t *testing.T, got, want []netip.Prefix) {
	if len(got) != len(want) {
		t.Errorf("got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
		return
	}
	for i, p := range got {
		if p != want[i] {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}

}

func TestPrefixSetSubtract(t *testing.T) {
	tests := []struct {
		set      []netip.Prefix
		subtract netip.Prefix
		want     []netip.Prefix
	}{
		{pfxs(), netip.Prefix{}, pfxs()},
		{pfxs("::0/1"), pfx("::0/1"), pfxs()},
		{pfxs("::0/2"), pfx("::0/2"), pfxs()},
		{pfxs("::0/128"), pfx("::0/128"), pfxs()},
		{pfxs("::0/128"), pfx("::0/127"), pfxs()},
		{pfxs("::0/128"), pfx("::1/128"), pfxs("::0/128")},
		{pfxs("::0/127"), pfx("::0/128"), pfxs("::1/128")},
		{pfxs("::2/127"), pfx("::3/128"), pfxs("::2/128")},
		{pfxs("::0/126"), pfx("::0/128"), pfxs("::1/128", "::2/127")},
		{pfxs("::0/126"), pfx("::3/128"), pfxs("::0/127", "::2/128")},
		// IPv4
		{
			set:      pfxs("1.2.3.0/30"),
			subtract: pfx("1.2.3.0/32"),
			want:     pfxs("1.2.3.1/32", "1.2.3.2/31"),
		},
	}
	for _, tt := range tests {
		pmb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			pmb.Add(p)
		}
		pmb.Subtract(tt.subtract)
		checkPrefixSlice(t, pmb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSetSubtractSet(t *testing.T) {
	tests := []struct {
		set      []netip.Prefix
		subtract []netip.Prefix
		want     []netip.Prefix
	}{
		{pfxs("::0/1"), pfxs("::0/1"), pfxs()},
		{pfxs("::0/2"), pfxs("::0/2"), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128")},
		{pfxs("::0/127"), pfxs("::0/128"), pfxs("::1/128")},
		{pfxs("::2/127"), pfxs("::3/128"), pfxs("::2/128")},
		{pfxs("::0/126"), pfxs("::0/128"), pfxs("::1/128", "::2/127")},
		{pfxs("::0/126"), pfxs("::3/128"), pfxs("::0/127", "::2/128")},
		{pfxs("::0/127"), pfxs("::0/128", "::1/128"), pfxs()},
		{pfxs("::3/128"), pfxs("::2/127"), pfxs()},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128"), pfxs("::1/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128", "::1/128"), pfxs()},
		{pfxs("::0/127", "::1/128"), pfxs("::0/127"), pfxs()},
		{pfxs("::3/128"), pfxs("::2/127", "::1/128"), pfxs()},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		subPsb := &PrefixSetBuilder{}
		for _, p := range tt.subtract {
			subPsb.Add(p)
		}
		psb.SubtractSet(subPsb.PrefixSet())
		checkPrefixSlice(t, psb.PrefixSet().Prefixes(), tt.want)
	}
}

func TestPrefixSetPrefixes(t *testing.T) {
	tests := []struct {
		add    []netip.Prefix
		remove []netip.Prefix
		want   []netip.Prefix
	}{
		{pfxs(), pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs(), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/128"), pfxs()},
		{pfxs("::0/128"), pfxs("::1/128"), pfxs("::0/128")},
		{pfxs("::0/128"), pfxs("::0/127"), pfxs("::0/128")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		for _, p := range tt.remove {
			psb.Remove(p)
		}
		ps := psb.PrefixSet()
		checkPrefixSlice(t, ps.Prefixes(), tt.want)
	}
}

func TestPrefixSetPrefixesCompact(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/127", "::0/128"), pfxs("::0/127")},
		{pfxs("::0/126", "::0/127"), pfxs("::0/126")},
		{pfxs("::0/1", "::0/128"), pfxs("::0/1")},
		{pfxs("0::0/127", "::0/128", "::1/128"), pfxs("::0/127")},
		{pfxs("0::0/127", "::0/128", "::2/128"), pfxs("::0/127", "::2/128")},
		{pfxs("1.2.3.0/24", "1.2.3.4/32"), pfxs("1.2.3.0/24")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		checkPrefixSlice(t, ps.PrefixesCompact(), tt.want)
	}
}
