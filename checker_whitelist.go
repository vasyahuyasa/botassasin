package main

import (
	"fmt"
	"log"
	"net"
)

var _ checker = &whitelistChecker{}

type whitelistConfig struct {
	IPs []string
}

type whitelistChecker struct {
	nets []*net.IPNet
}

func newWhitelistChecker(cfg whitelistConfig) (*whitelistChecker, error) {
	var nets []*net.IPNet

	for _, cidr := range cfg.IPs {
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %v", cidr, err)
		}

		log.Println(ip, ipnet)

		nets = append(nets, ipnet)
	}

	log.Print("created whitrlist with %d rules", len(nets))

	return &whitelistChecker{
		nets: nets,
	}, nil
}

func (wl *whitelistChecker) Check(l logLine) (score harmScore, descision instantDecision) {
	for _, ipnet := range wl.nets {
		if ipnet.Contains(l.IP()) {
			return 0, decisionWhitelist
		}
	}

	return 0, decisionNone
}
