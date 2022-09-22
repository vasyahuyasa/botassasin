package main

import (
	"net"
	"net/netip"
	"testing"

	"github.com/yl2chen/cidranger"
	"go4.org/netipx"
)

func BenchmarkIpListIn(b *testing.B) {
	list := ipList{
		ips:    benchmarkIpListDataProvider(),
		action: listCheckerActionBlock,
	}

	ip := net.IPv4(20, 20, 101, 202)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		list.contains(ip)
	}
}

func BenchmarkIpList2In(b *testing.B) {
	ranger := cidranger.NewPCTrieRanger()

	for _, ipnet := range benchmarkIpListDataProvider() {
		err := ranger.Insert(cidranger.NewBasicRangerEntry(*ipnet))
		if err != nil {
			b.Fatal(err)
		}
	}

	list := ipList2{
		ranger: ranger,
	}

	ip := net.IPv4(20, 20, 101, 202)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		list.contains(ip)
	}
}

func BenchmarkIpList3In(b *testing.B) {
	ipSetBuilder := netipx.IPSetBuilder{}

	for _, ipnet := range benchmarkIpListDataProvider() {
		p := netip.MustParsePrefix(ipnet.String())
		ipSetBuilder.AddPrefix(p)
	}

	ipset, err := ipSetBuilder.IPSet()
	if err != nil {
		b.Fatal(err)
	}

	list := ipList3{
		ipset: ipset,
	}

	ip := netip.AddrFrom4([4]byte{20, 20, 101, 202})

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		list.contains(ip)
	}
}

func benchmarkIpListDataProvider() []*net.IPNet {
	return []*net.IPNet{
		{
			IP:   net.IPv4(1, 2, 3, 4),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 5),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 6),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 7),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 8),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 9),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 10),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(1, 2, 3, 11),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		{
			IP:   net.IPv4(10, 20, 30, 40),
			Mask: net.IPv4Mask(255, 255, 0, 0),
		},
	}
}
