package main

import (
	_ "embed"
	"net"
	"testing"
)

func Test_geoIPChecker_Check(t *testing.T) {
	tests := []struct {
		name         string
		cfg          geoIPConfig
		logLine      logLine
		wantScore    harmScore
		wantDecision instantDecision
	}{
		{
			name: "local db RU allow",
			cfg: geoIPConfig{
				Path:             "test-data/geoip2-country.mmdb",
				AllowedCountries: []string{"RU"},
			},
			logLine: logLine{
				ip: net.IPv4(95, 173, 136, 72),
			},
			wantScore:    0,
			wantDecision: decisionWhitelist,
		},
		{
			name: "local db RU ban",
			cfg: geoIPConfig{
				Path:             "test-data/geoip2-country.mmdb",
				AllowedCountries: []string{"FR"},
			},
			logLine: logLine{
				ip: net.IPv4(95, 173, 136, 72),
			},
			wantScore:    0,
			wantDecision: decisionBan,
		},
		{
			name: "local db unknown ban",
			cfg: geoIPConfig{
				Path:             "test-data/geoip2-country.mmdb",
				AllowedCountries: []string{"FR", "RU", "CN"},
			},
			logLine: logLine{
				ip: net.IPv4(1, 2, 3, 4),
			},
			wantScore:    0,
			wantDecision: decisionBan,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gi, err := newGeoIPChecker(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			gotScore, gotDecision := gi.Check(&tt.logLine)
			if gotScore != tt.wantScore {
				t.Errorf("geoIPChecker.Check() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
			if gotDecision != tt.wantDecision {
				t.Errorf("geoIPChecker.Check() gotDecision = %v, want %v", gotDecision, tt.wantDecision)
			}
		})
	}
}
