package netipmap

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
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		if got := ps.OverlapsPrefix(tt.get); got != tt.want {
			t.Errorf("pm.OverlapsPrefix(%s) = %v, want %v", tt.get, got, tt.want)
		}
	}
}

func checkPrefixSlice(t *testing.T, got, want []netip.Prefix) {
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
		return
	}
	for i, p := range got {
		if p != want[i] {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}

}

func TestPrefixSetSubtractFromPrefix(t *testing.T) {
	tests := []struct {
		set  []netip.Prefix
		get  netip.Prefix
		want []netip.Prefix
	}{
		//{pfxs(), pfx("::0/128"), pfxs("::0/128")},
		//{pfxs("::0/128"), pfx("::0/128"), pfxs()},
		//{pfxs("::0/128"), pfx("::1/128"), pfxs("::1/128")},
		//{pfxs("::0/128"), pfx("::0/127"), pfxs("::1/128")},
		//{pfxs("::0/127"), pfx("::0/128"), pfxs()},
		//{pfxs("::0/128", "::1/128"), pfx("::2/128"), pfxs("::2/128")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.set {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		got := ps.SubtractFromPrefix(tt.get)
		checkPrefixSlice(t, got.Prefixes(), tt.want)
	}
}
