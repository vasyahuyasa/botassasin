package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

var _ checker = &iPwhitelistChecker{}
var ipMaskFull = net.IPMask{0xff, 0xff, 0xff, 0xff}

type iPwhitelistConfig struct {
	Allow []string
}

type iPwhitelistChecker struct {
	nets []*net.IPNet
}

func newWhitelistChecker(cfg iPwhitelistConfig) (*iPwhitelistChecker, error) {
	var nets []*net.IPNet

	for _, addr := range cfg.Allow {
		ipnet, err := parseIPorCIDR(addr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %v", addr, err)
		}

		nets = append(nets, ipnet)
	}

	log.Printf("whitelist created with %d rules", len(nets))

	return &iPwhitelistChecker{
		nets: nets,
	}, nil
}

func (wl *iPwhitelistChecker) Check(l logLine) (score harmScore, descision instantDecision) {
	for _, ipnet := range wl.nets {
		if ipnet.Contains(l.IP()) {
			return 0, decisionWhitelist
		}
	}

	return 0, decisionNone
}

func parseIPorCIDR(ip string) (*net.IPNet, error) {
	if strings.IndexByte(ip, '/') != -1 {
		_, ipNet, err := net.ParseCIDR(ip)
		return ipNet, err
	}

	ip4 := net.ParseIP(ip)

	return &net.IPNet{IP: ip4, Mask: ipMaskFull}, nil
}
