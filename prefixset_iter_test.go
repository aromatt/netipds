//go:build go1.23

package netipds

import (
	"iter"
	"net/netip"
	"slices"
	"testing"
)

func TestPrefixSetAll6(t *testing.T) {
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
		{pfxs("8000::/1"), pfxs("8000::/1")},
		{pfxs("::0/1", "8000::/1"), pfxs("::0/1", "8000::/1")},
		{pfxs("0::0/127", "::0/128", "::1/128"), pfxs("0::0/127", "::0/128", "::1/128")},
		{pfxs("0::0/127", "::0/128", "::2/128"), pfxs("0::0/127", "::0/128", "::2/128")},
	}
	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()

		seqAll6 := ps.All6()
		checkPrefixSeq(t, seqAll6, tt.want)
		checkYieldFalse(t, seqAll6)

		// All should yield the same items since only IPv6s were added
		seqAll := ps.All()
		checkPrefixSeq(t, seqAll, tt.want)
		checkYieldFalse(t, seqAll)
	}
}

func TestPrefixSetAll4(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32", "1.2.3.5/32"), pfxs("1.2.3.4/32", "1.2.3.5/32")},
		{pfxs("1.2.3.4/31", "1.2.3.4/32"), pfxs("1.2.3.4/31", "1.2.3.4/32")},
		{pfxs("1.2.3.4/30", "1.2.3.4/31"), pfxs("1.2.3.4/30", "1.2.3.4/31")},
		{pfxs("0.0.0.0/1", "1.2.3.4/32"), pfxs("0.0.0.0/1", "1.2.3.4/32")},
		{pfxs("128.0.0.0/1"), pfxs("128.0.0.0/1")},
		{pfxs("0.0.0.0/1", "128.0.0.0/1"), pfxs("0.0.0.0/1", "128.0.0.0/1")},
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.5/32"),
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.5/32"),
		},
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.6/32"),
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.6/32"),
		},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()

		seqAll4 := ps.All4()
		checkPrefixSeq(t, seqAll4, tt.want)
		checkYieldFalse(t, seqAll4)
	}
}

func TestPrefixSetAll(t *testing.T) {
	tests := []struct {
		add4 []netip.Prefix
		add6 []netip.Prefix
		want []netip.Prefix
	}{
		// no IPv4, no IPv6
		{nil, nil, nil},

		// IPv4 only
		{
			pfxs("1.2.3.4/32", "128.0.0.0/1"),
			nil,
			pfxs("1.2.3.4/32", "128.0.0.0/1"),
		},

		// IPv6 only
		{
			nil,
			pfxs("::0/128", "::1/128"),
			pfxs("::0/128", "::1/128"),
		},

		// mixed families
		{
			pfxs("1.2.3.4/32", "128.0.0.0/1"),
			pfxs("::0/128", "8000::/1"),
			append(
				pfxs("1.2.3.4/32", "128.0.0.0/1"),
				pfxs("::0/128", "8000::/1")...,
			),
		},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add4 {
			psb.Add(p)
		}
		for _, p := range tt.add6 {
			psb.Add(p)
		}
		ps := psb.PrefixSet()

		seqAll := ps.All()
		checkPrefixSeq(t, seqAll, tt.want)
		checkYieldFalse(t, seqAll)
	}
}

func TestPrefixSetAllCompact4(t *testing.T) {
	tests := []struct {
		add  []netip.Prefix
		want []netip.Prefix
	}{
		{pfxs(), pfxs()},
		{pfxs("1.2.3.4/32"), pfxs("1.2.3.4/32")},
		{pfxs("1.2.3.4/32", "1.2.3.5/32"), pfxs("1.2.3.4/32", "1.2.3.5/32")},
		{pfxs("1.2.3.4/31", "1.2.3.4/32"), pfxs("1.2.3.4/31")},
		{pfxs("1.2.3.4/30", "1.2.3.4/31"), pfxs("1.2.3.4/30")},
		{pfxs("0.0.0.0/1", "1.2.3.4/32"), pfxs("0.0.0.0/1")},
		{pfxs("128.0.0.0/1"), pfxs("128.0.0.0/1")},
		{pfxs("0.0.0.0/1", "128.0.0.0/1"), pfxs("0.0.0.0/1", "128.0.0.0/1")},
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.5/32"),
			pfxs("1.2.3.4/31"),
		},
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32", "1.2.3.6/32"),
			pfxs("1.2.3.4/31", "1.2.3.6/32"),
		},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add {
			psb.Add(p)
		}
		ps := psb.PrefixSet()

		seq4 := ps.AllCompact4()
		checkPrefixSeq(t, seq4, tt.want)
		checkYieldFalse(t, seq4)
	}
}

func TestPrefixSetAllCompact(t *testing.T) {
	tests := []struct {
		add4 []netip.Prefix
		add6 []netip.Prefix
		want []netip.Prefix
	}{
		// no IPv4, no IPv6
		{nil, nil, nil},

		// IPv4 only
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32"),
			nil,
			pfxs("1.2.3.4/31"),
		},

		// IPv6 only
		{
			nil,
			pfxs("::0/127", "::0/128", "::2/128"),
			pfxs("::0/127", "::2/128"),
		},

		// mixed families
		{
			pfxs("1.2.3.4/31", "1.2.3.4/32"),
			pfxs("::0/127", "::0/128", "::2/128"),
			append(
				pfxs("1.2.3.4/31"),
				pfxs("::0/127", "::2/128")...,
			),
		},
	}

	for _, tt := range tests {
		psb := &PrefixSetBuilder{}
		for _, p := range tt.add4 {
			psb.Add(p)
		}
		for _, p := range tt.add6 {
			psb.Add(p)
		}
		ps := psb.PrefixSet()

		seqAll := ps.AllCompact()
		checkPrefixSeq(t, seqAll, tt.want)
		checkYieldFalse(t, seqAll)
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
