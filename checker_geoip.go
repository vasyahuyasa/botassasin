package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/oschwald/maxminddb-golang"
	"github.com/vasyahuyasa/botassasin/log"
)

const countryField = "country"

//go:embed GeoLite2-Country.mmdb
var embededGeoIP []byte

var _ checker = &geoIPChecker{}

type geoIPRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

type geoIPConfig struct {
	Path             string   `yaml:"path"`
	AllowedCountries []string `yaml:"allowed_countries"`
}

type geoIPChecker struct {
	db               *maxminddb.Reader
	allowedCountries []string
}

func newGeoIPChecker(cfg geoIPConfig) (*geoIPChecker, error) {
	var db *maxminddb.Reader
	var dbErr error
	var err error

	if cfg.Path == "" {
		db, dbErr = maxminddb.FromBytes(embededGeoIP)
		if dbErr != nil {
			err = fmt.Errorf("cannot load GeoIP database from %q: %w", cfg.Path, dbErr)
		}
	} else {
		db, dbErr = maxminddb.Open(cfg.Path)
		if dbErr != nil {
			err = fmt.Errorf("cannot load embeded GeoIP database: %w", dbErr)
		}
	}

	if err != nil {
		return nil, err
	}

	dbPath := "embeded"
	if cfg.Path != "" {
		dbPath = cfg.Path
	}

	log.Printf("geoIP loaded %q, allow countries %s", dbPath, strings.Join(cfg.AllowedCountries, ","))

	return &geoIPChecker{
		db:               db,
		allowedCountries: cfg.AllowedCountries,
	}, nil
}

func (gi *geoIPChecker) Check(l *logLine) (score harmScore, decision instantDecision) {
	var rec geoIPRecord

	err := gi.db.Lookup(l.IP(), &rec)
	if err != nil {
		log.Printf("cannot check country for %v: %v", l, err)
		return 0, decisionNone
	}

	l.Set(countryField, rec.Country.ISOCode)

	for _, allowed := range gi.allowedCountries {
		if rec.Country.ISOCode == allowed {
			return 0, decisionWhitelist
		}
	}

	return 0, decisionBan
}
