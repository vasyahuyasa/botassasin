package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/vasyahuyasa/botassasin/log"
)

const (
	resolverLookupTimeout = time.Second * 5
	dnsDialerTimeout      = time.Second * 5
)

type resolverListConfig struct {
	addrs []string
}

type reverseDNSCheckerConfig struct {
	Rules []struct {
		Field          string             `yaml:"field"`
		FieldContains  []string           `yaml:"field_contains"`
		DomainSuffixes []string           `yaml:"domain_suffixes"`
		Resolvers      resolverListConfig `yaml:"resolver"`
	} `yaml:"rules"`
}

type resolverPool struct {
	// next index in pool
	next int

	resolvers []*net.Resolver
}

type reverseDNSCheckerRule struct {
	field         string
	fieldContains []string
	domainSufixes []string
	resolverPool  *resolverPool
}

type reverseDNSChecker struct {
	rules []reverseDNSCheckerRule
}

func newReverseDNSChecker(cfg reverseDNSCheckerConfig) (*reverseDNSChecker, error) {
	var rules []reverseDNSCheckerRule

	for _, r := range cfg.Rules {
		if r.Field == "" {
			return nil, fmt.Errorf("field cannot be empty")
		}

		if len(r.FieldContains) == 0 {
			return nil, fmt.Errorf("field_contains cannot be empty")
		}

		if len(r.DomainSuffixes) == 0 {
			return nil, fmt.Errorf("domain_suffixes cannot be empty")
		}

		resolverPool := makeResolverPoolFromConfig(r.Resolvers)

		log.Printf("reverse dns field %q must contains [%s] DNS suffix [%s] resolver %s", r.Field, strings.Join(r.FieldContains, ","), strings.Join(r.DomainSuffixes, ","), r.Resolvers)

		rules = append(rules, reverseDNSCheckerRule{
			field:         r.Field,
			fieldContains: r.FieldContains,
			domainSufixes: r.DomainSuffixes,
			resolverPool:  resolverPool,
		})
	}

	return &reverseDNSChecker{rules: rules}, nil
}

func makeResolverPoolFromConfig(resolversCfg resolverListConfig) *resolverPool {
	if resolversCfg.len() == 0 {
		return newDefaultResolverPool()
	}

	var resolvers []*net.Resolver

	for _, addr := range resolversCfg.addrs {
		// default dns port
		if !strings.Contains(addr, ":") {
			addr += ":53"
		}

		resolvers = append(resolvers, &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: dnsDialerTimeout,
				}

				switch network {
				case "udp", "udp4", "udp6":
					return d.DialContext(ctx, "udp4", addr)
				case "tcp", "tcp4", "tcp6":
					return d.DialContext(ctx, "tcp4", addr)
				default:
					panic("PatchNet.Dial: unknown network")
				}
			},
		})
	}

	return newResolverPool(resolvers)
}

func (rdns *reverseDNSChecker) Check(l *logLine) (score harmScore, descision instantDecision) {
	for _, rule := range rdns.rules {
		if rule.match(*l) {
			ok, err := rule.fineDNS(l.IP())
			if err != nil {
				log.Printf("cannot check DNS for %q: %v", l.IP(), err)
				return 0, decisionNone
			}

			if ok {
				return 0, decisionWhitelist
			}

			return 0, decisionBan
		}
	}

	return 0, decisionNone
}

func (r *reverseDNSCheckerRule) match(l logLine) bool {
	field, ok := l.Get(r.field)
	if !ok {
		return false
	}

	for _, substr := range r.fieldContains {
		if strings.Contains(field, substr) {
			return true
		}
	}

	return false
}

func (r *reverseDNSCheckerRule) hasAlloweedDNSSuffix(name string) bool {
	for _, suffix := range r.domainSufixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	return false
}

func (r *reverseDNSCheckerRule) fineDNS(ip net.IP) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), resolverLookupTimeout)
	defer cancel()

	// reverse DNS lookup
	resolver := r.resolverPool.get()

	addrs, err := resolver.LookupAddr(ctx, ip.String())

	if err != nil {
		// any misconfigured DNS lead to ban
		dnsErr := &net.DNSError{}
		if errors.As(err, &dnsErr) {
			log.Printf("reverse lookup error: %v", dnsErr)
			return false, nil
		}

		return false, fmt.Errorf("reverse lookup %q failed: %w", ip, err)
	}

	// forward DNS lookup by name
	for _, a := range addrs {
		name := strings.TrimRight(a, ".")

		lookupCtx, lookupCancel := context.WithTimeout(context.Background(), resolverLookupTimeout)
		defer lookupCancel()

		if r.hasAlloweedDNSSuffix(name) {
			lookupIPs, lookupErr := resolver.LookupIPAddr(lookupCtx, name)
			if lookupErr != nil {
				dnsErr := &net.DNSError{}
				if errors.As(err, &dnsErr) {
					log.Printf("lookup error: %v", dnsErr)
					return false, nil
				}

				return false, fmt.Errorf("cannot lookup %q: %w", name, lookupErr)
			}

			for _, lookupIP := range lookupIPs {
				if lookupIP.IP.Equal(ip) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (list *resolverListConfig) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	addr, ok := list.parseAsString(data)
	if ok {
		list.addrs = []string{
			addr,
		}

		return nil
	}

	addrs, err := list.parseAsStringSlice(data)
	if err != nil {
		return err
	}

	list.addrs = addrs

	return nil
}

func (list *resolverListConfig) parseAsString(data [](byte)) (string, bool) {
	var str string

	err := json.Unmarshal(data, &str)
	if err != nil {
		return "", false
	}

	return str, true
}

func (list *resolverListConfig) parseAsStringSlice(data [](byte)) ([]string, error) {
	var addrs []string

	err := json.Unmarshal(data, &addrs)
	if err != nil {
		return nil, err
	}

	return addrs, nil
}

func (list *resolverListConfig) len() int {
	return len(list.addrs)
}

func newResolverPool(resolvers []*net.Resolver) *resolverPool {
	if len(resolvers) == 0 {
		log.Fatalf("resolvers list must contain at last one resolver")
	}

	return &resolverPool{
		resolvers: resolvers,
	}
}

func newDefaultResolverPool() *resolverPool {
	return &resolverPool{
		resolvers: []*net.Resolver{
			&net.Resolver{},
		},
	}
}

func (pool *resolverPool) get() *net.Resolver {
	r := pool.resolvers[pool.next]

	pool.next++
	if pool.next > len(pool.resolvers) {
		pool.next = 0
	}

	return r
}
