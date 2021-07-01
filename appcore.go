package main

import (
	"log"
	"net"
)

type appcore struct {
	passCache  ipCache
	blockCache ipCache

	streamer *logStreamer
	c        *chain
	act      *action
}

type ipCache map[string]struct{}

func newAppCore(streamer *logStreamer, c *chain, act *action) *appcore {
	return &appcore{
		passCache:  ipCache{},
		blockCache: ipCache{},

		streamer: streamer,
		c:        c,
		act:      act,
	}
}

func (core *appcore) run() error {
	for l := range core.streamer.C() {
		ip := l.IP()

		if core.passCache.contains(ip) {
			log.Println(ip.String(), "in whitelist")
			continue
		}

		if core.blockCache.contains(ip) {
			log.Println(ip.String(), "in blocklist")
			continue
		}

		if core.c.NeedBan(l) {
			core.blockCache.add(ip)

			err := core.act.Execute(l)
			if err != nil {
				log.Printf("cannot execute action: %v", err)
			}
			continue
		}

		core.passCache.add(ip)
	}

	return core.streamer.Err()
}

func (c ipCache) contains(ip net.IP) bool {
	_, ok := c[ip.String()]

	return ok
}

func (c ipCache) add(ip net.IP) {
	c[ip.String()] = struct{}{}
}
