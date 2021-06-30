package main

import (
	"io"
	"net"
	"testing"

	"log"

	"github.com/foxcpp/go-mockdns"
)

var dnsTestZone = map[string]mockdns.Zone{
	"bot.unittesting.org.": {
		A: []string{"1.2.3.4"},
	},
	"4.3.2.1.in-addr.arpa.": {
		PTR: []string{"bot.unittesting.org."},
	},
	"crawl-66-249-66-1.googlebot.com.": {
		A: []string{"66.249.66.1"},
	},
	"1.66.249.66.in-addr.arpa.": {
		PTR: []string{"crawl-66-249-66-1.googlebot.com."},
	},
	"harmbot.unittesting.org.": {
		A: []string{"2.3.4.5"},
	},
	"5.4.3.2.in-addr.arpa.": {
		PTR: []string{"."},
	},
	"googlefakebot.unittesting.org.": {
		A: []string{"3.4.5.6"},
	},
	"6.5.4.3.in-addr.arpa.": {
		PTR: []string{"fake-66-249-66-1.googlebot.com."},
	},
}

func Test_reverseDNSChecker_Check(t *testing.T) {
	createDNSServer, close := dnsMockCreator(t)
	defer close()

	tests := []struct {
		name          string
		cfg           reverseDNSCheckerConfig
		logLine       logLine
		wantScore     harmScore
		wantDescision instantDecision
	}{
		{
			name: "simple whitelist",
			cfg: reverseDNSCheckerConfig{
				Rules: []struct {
					Field          string   `yaml:"field"`
					FieldContains  []string `yaml:"field_contains"`
					DomainSuffixes []string `yaml:"domain_suffixes"`
					ResolverAddr   string   `yaml:"resolver"`
				}{
					{
						Field:          "user_agent",
						FieldContains:  []string{"clientbot"},
						DomainSuffixes: []string{"unittesting.org"},
						ResolverAddr:   createDNSServer(dnsTestZone),
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(1, 2, 3, 4),
				fields: map[string]string{
					"user_agent": "unit_go_clientbot",
				},
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "googlebot whitelist",
			cfg: reverseDNSCheckerConfig{
				Rules: []struct {
					Field          string   `yaml:"field"`
					FieldContains  []string `yaml:"field_contains"`
					DomainSuffixes []string `yaml:"domain_suffixes"`
					ResolverAddr   string   `yaml:"resolver"`
				}{
					{
						Field:          "user_agent",
						FieldContains:  []string{"Googlebot"},
						DomainSuffixes: []string{"googlebot.com", "google.com"},
						ResolverAddr:   createDNSServer(dnsTestZone),
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(66, 249, 66, 1),
				fields: map[string]string{
					"user_agent": "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; +http://www.google.com/bot.html) Chrome/W.X.Y.Z Safari/537.36",
				},
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "not exist PTR blacklist",
			cfg: reverseDNSCheckerConfig{
				Rules: []struct {
					Field          string   `yaml:"field"`
					FieldContains  []string `yaml:"field_contains"`
					DomainSuffixes []string `yaml:"domain_suffixes"`
					ResolverAddr   string   `yaml:"resolver"`
				}{
					{
						Field:          "user_agent",
						FieldContains:  []string{"clientbot"},
						DomainSuffixes: []string{"unittesting.org"},
						ResolverAddr:   createDNSServer(dnsTestZone),
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(9, 9, 9, 9),
				fields: map[string]string{
					"user_agent": "unit_go_clientbot",
				},
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "bad PTR blacklist",
			cfg: reverseDNSCheckerConfig{
				Rules: []struct {
					Field          string   `yaml:"field"`
					FieldContains  []string `yaml:"field_contains"`
					DomainSuffixes []string `yaml:"domain_suffixes"`
					ResolverAddr   string   `yaml:"resolver"`
				}{
					{
						Field:          "user_agent",
						FieldContains:  []string{"clientbot"},
						DomainSuffixes: []string{"googlebot.com."},
						ResolverAddr:   createDNSServer(dnsTestZone),
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(2, 3, 4, 5),
				fields: map[string]string{
					"user_agent": "unit_go_clientbot",
				},
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "fake PTR blacklist",
			cfg: reverseDNSCheckerConfig{
				Rules: []struct {
					Field          string   `yaml:"field"`
					FieldContains  []string `yaml:"field_contains"`
					DomainSuffixes []string `yaml:"domain_suffixes"`
					ResolverAddr   string   `yaml:"resolver"`
				}{
					{
						Field:          "user_agent",
						FieldContains:  []string{"googlebot"},
						DomainSuffixes: []string{"googlebot.com."},
						ResolverAddr:   createDNSServer(dnsTestZone),
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(3, 4, 5, 6),
				fields: map[string]string{
					"user_agent": "testing googlebot",
				},
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdns, err := newReverseDNSChecker(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			gotScore, gotDescision := rdns.Check(tt.logLine)
			if gotScore != tt.wantScore {
				t.Errorf("reverseDNSChecker.Check() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
			if gotDescision != tt.wantDescision {
				t.Errorf("reverseDNSChecker.Check() gotDescision = %v, want %v", gotDescision, tt.wantDescision)
			}
		})
	}
}

func dnsMockCreator(t *testing.T) (createDNS func(map[string]mockdns.Zone) string, close func()) {
	var servers []*mockdns.Server

	return func(zones map[string]mockdns.Zone) string {
			srv, err := mockdns.NewServerWithLogger(zones, log.New(io.Discard, "", log.LstdFlags), false)
			if err != nil {
				t.Fatal(err)
			}

			servers = append(servers, srv)

			return srv.LocalAddr().String()
		}, func() {
			for _, srv := range servers {
				srv.Close()
			}
		}
}
