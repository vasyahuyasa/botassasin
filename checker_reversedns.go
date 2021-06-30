package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	resolverLookupTimeout = time.Second * 5
	dnsDialerTimeout      = time.Second * 5
)

type reverseDNSCheckerConfig struct {
	Rules []struct {
		Field          string   `yaml:"field"`
		FieldContains  []string `yaml:"field_contains"`
		DomainSuffixes []string `yaml:"domain_suffixes"`
		ResolverAddr   string   `yaml:"resolver"`
	} `yaml:"rules"`
}

type reverseDNSCheckerRule struct {
	field         string
	fieldContains []string
	domainSufixes []string
	resolver      *net.Resolver
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

		var resolver *net.Resolver

		if r.ResolverAddr != "" {
			resolver = &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: dnsDialerTimeout,
					}

					switch network {
					case "udp", "udp4", "udp6":
						return d.DialContext(ctx, "udp4", r.ResolverAddr)
					case "tcp", "tcp4", "tcp6":
						return d.DialContext(ctx, "tcp4", r.ResolverAddr)
					default:
						panic("PatchNet.Dial: unknown network")
					}
				},
			}
		}

		log.Printf("reverse dns field %q contains [%s] DNS suffix [%s] resolver %s", r.Field, strings.Join(r.FieldContains, ","), strings.Join(r.DomainSuffixes, ","), r.ResolverAddr)

		rules = append(rules, reverseDNSCheckerRule{
			field:         r.Field,
			fieldContains: r.FieldContains,
			domainSufixes: r.DomainSuffixes,
			resolver:      resolver,
		})
	}

	return &reverseDNSChecker{rules: rules}, nil
}

func (rdns *reverseDNSChecker) Check(l logLine) (score harmScore, descision instantDecision) {
	for _, rule := range rdns.rules {
		if rule.match(l) {
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
	addrs, err := r.resolver.LookupAddr(ctx, ip.String())

	if err != nil {
		// any misconfigured DNS lead to ban
		dnsErr := &net.DNSError{}
		if errors.As(err, &dnsErr) {
			log.Printf("dns error: %v", dnsErr)
			return false, nil
		}

		return false, fmt.Errorf("cannot lookup addr %q: %w", ip, err)
	}

	// forward DNS lookup by name
	for _, a := range addrs {
		name := strings.TrimRight(a, ".")

		lookupCtx, lookupCancel := context.WithTimeout(context.Background(), resolverLookupTimeout)
		defer lookupCancel()

		if r.hasAlloweedDNSSuffix(name) {
			lookupIPs, lookupErr := r.resolver.LookupIPAddr(lookupCtx, name)
			if lookupErr != nil {
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
