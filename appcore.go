package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/vasyahuyasa/botassasin/log"
)

const (
	cacheWriteTimeFormat = time.RFC3339
	saveInterval         = time.Minute
)

type appcore struct {
	passCache  *ipCache
	blockCache *ipCache

	streamer *logStreamer
	c        *chain
	act      *action
	log      *logPrinter
}

type ipCache struct {
	path string
	mu   *sync.RWMutex
	data map[string]struct {
		added time.Time
	}
}

func newAppCore(streamer *logStreamer, c *chain, act *action, lp *logPrinter, cachepath string) *appcore {
	passCache, err := newIPCaheFromFile(cachepath)
	if err != nil {
		log.Printf("cannot load cache file %s: %v", cachepath, err)
		passCache = newIPCache(cachepath)
	}

	return &appcore{
		passCache:  passCache,
		blockCache: newIPCache(""),

		streamer: streamer,
		c:        c,
		act:      act,
		log:      lp,
	}
}

// TODO: replace string with function that return writer
func newIPCache(path string) *ipCache {
	return &ipCache{
		path: path,
		mu:   &sync.RWMutex{},
		data: map[string]struct {
			added time.Time
		}{},
	}
}

func newIPCaheFromFile(path string) (*ipCache, error) {
	if path == "" {
		return newIPCache(path), nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open cache file %q: %w", path, err)
	}

	scanner := bufio.NewScanner(f)

	cache := newIPCache(path)

	for scanner.Scan() {
		str := scanner.Text()

		var strIP, strTime string

		_, err := fmt.Sscanf(str, "%s %s", &strIP, &strTime)
		if err != nil {
			return nil, fmt.Errorf("cannot parse cache line %q: %w", str, err)
		}

		t, err := time.Parse(cacheWriteTimeFormat, strTime)
		if err != nil {
			return nil, fmt.Errorf("cannot parse time %q: %w", strTime, err)
		}

		ip := net.ParseIP(strIP)

		cache.addWithTime(ip, t)
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("cannot read cache: %w", scanner.Err())
	}

	log.Printf("%d records loaded from cache %s", len(cache.data), path)

	return cache, nil
}

func (core *appcore) run() error {
	go core.passCache.saver()

	for l := range core.streamer.C() {
		ip := l.IP()

		if core.passCache.Contains(ip) {
			log.Debugf("%s in whitelist", ip.String())
			continue
		}

		if core.blockCache.Contains(ip) {
			log.Debugf("%s in blocklist", ip.String())
			continue
		}

		if core.c.NeedBan(l) {
			core.blockCache.Add(ip)

			core.log.Println(*l)

			err := core.act.Execute(*l)
			if err != nil {
				log.Printf("cannot execute action: %v", err)
			}
			continue
		}

		core.passCache.Add(ip)
	}

	return core.streamer.Err()
}

func (c *ipCache) Contains(ip net.IP) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.data[ip.String()]

	return ok
}

func (c *ipCache) Add(ip net.IP) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.addWithTime(ip, time.Now())
}

func (c *ipCache) addWithTime(ip net.IP, t time.Time) {
	c.data[ip.String()] = struct{ added time.Time }{added: t}
}

func (c *ipCache) writeTo(w io.Writer) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for ip, stat := range c.data {
		_, err := fmt.Fprintf(w, "%s %s\n", ip, stat.added.Format(cacheWriteTimeFormat))
		if err != nil {
			return 0, fmt.Errorf("cannot save cache: %w", err)
		}
	}

	return len(c.data), nil
}

func (c *ipCache) saver() {
	if c.path == "" {
		return
	}

	ticker := time.NewTicker(saveInterval)

	for range ticker.C {
		f, err := os.Create(c.path)
		if err != nil {
			log.Printf("can not open file for save cache: %v", err)
			continue
		}

		count, err := c.writeTo(f)
		if err != nil {
			log.Printf("cannot write cache to file: %v", err)
			f.Close()
			continue
		}

		f.Close()
		log.Printf("cache saved %d records", count)
	}
}
