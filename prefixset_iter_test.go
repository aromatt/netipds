//go:build go1.23

package netipds

import (
	"iter"
	"net/netip"
	"slices"
	"testing"
)

func TestPrefixSetAll(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("::0/128"), pfxs("::0/128")},
		{pfxs("::0/128", "::1/128"), pfxs("::0/128", "::1/128")},
		{pfxs("::0/127", "::0/128"), pfxs("::0/127", "::0/128")},
		{pfxs("::0/126", "::0/127"), pfxs("::0/126", "::0/127")},
		{pfxs("::0/1", "::0/128"), pfxs("::0/1", "::0/128")},
		{pfxs("fc00::/7"), pfxs("fc00::/7")},
		{pfxs("::1/128", "fc00::/7", "fe80::/10", "ff00::/8"), pfxs("::1/128", "fc00::/7", "fe80::/10", "ff00::/8")},
		{pfxs("0::0/127", "::0/128", "::1/128"), pfxs("0::0/127", "::0/128", "::1/128")},
		{pfxs("0::0/127", "::0/128", "::2/128"), pfxs("0::0/127", "::0/128", "::2/128")},
		{pfxs("1.2.3.0/24", "1.2.3.4/32"), pfxs("1.2.3.0/24", "1.2.3.4/32")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()
		seq := ps.All()
		checkPrefixSeq(t, seq, tt.want)
		checkYieldFalse(t, seq)
	}
}

func TestPrefixSetAllCompact(t *testing.T) {
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
		{pfxs("fc00::/7"), pfxs("fc00::/7")},
		{pfxs("::1/128", "fc00::/7", "fe80::/10", "ff00::/8"), pfxs("::1/128", "fc00::/7", "fe80::/10", "ff00::/8")},
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
		seq := ps.AllCompact()
		checkPrefixSeq(t, seq, tt.want)
		checkYieldFalse(t, seq)
	}
}

func checkPrefixSeq(t *testing.T, seq iter.Seq[netip.Prefix], want []netip.Prefix) {
	t.Helper()
	got := slices.AppendSeq(make([]netip.Prefix, 0, len(want)), seq)
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func checkYieldFalse[T any](t *testing.T, seq iter.Seq[T]) {
	t.Helper()
	var i int
	for range seq {
		i++
		break
	}
	if i > 1 {
		t.Fatal("iteration continued after yield returned false")
	}
}
