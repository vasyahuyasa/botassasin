//go:generate stringer -type=instantDecision

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vasyahuyasa/botassasin/log"
	"gopkg.in/yaml.v2"
)

const (
	decisionNone instantDecision = iota
	decisionBan
	decisionWhitelist

	checkerField     = "checker"
	scoreField       = "score"
	scoreCheckerName = "score"
)

type instantDecision int

type harmScore int

type checker interface {
	Check(*logLine) (harm harmScore, descision instantDecision)
}

type checkerWithKind struct {
	checker
	kind string
}

type chain struct {
	reportFn reportCheckerWorkTime
	checkers []*checkerWithKind
}

type reportCheckerWorkTime func(name string, seconds float64)

func newChainFromConfig(cfg config, reportFn reportCheckerWorkTime) (*chain, error) {
	var checkers []*checkerWithKind

	for _, checkerCfg := range cfg.Checkers {
		c, err := checkerFromConfig(checkerCfg)
		if err != nil {
			return nil, fmt.Errorf("cannot create checker: %w", err)
		}

		checkers = append(checkers, c)
	}

	return &chain{
		reportFn: reportFn,
		checkers: checkers,
	}, nil
}

func (c *chain) NeedBan(l *logLine) bool {
	score := harmScore(0)

	for _, chk := range c.checkers {
		startedAt := time.Now()

		s, decision := chk.Check(l)

		c.reportFn(chk.kind, time.Since(startedAt).Seconds())

		log.Debugf("%s %s score: %d decision: %s", l.IP(), chk.kind, s, decision)

		if decision == decisionNone {
			score += s
			continue
		}

		l.Set(checkerField, chk.kind)
		l.Set(scoreField, strconv.Itoa(int(score)))

		return decision == decisionBan
	}

	log.Debugf("%s total score: %d", l.IP(), score)

	l.Set(checkerField, scoreCheckerName)
	l.Set(scoreField, strconv.Itoa(int(score)))

	return score > 0
}

func checkerFromConfig(cfg checkerConfig) (*checkerWithKind, error) {
	var kindOnly struct {
		Kind string
	}

	err := unmarshalConfig(cfg, &kindOnly)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal checker config: %w", err)
	}

	switch strings.ToLower(kindOnly.Kind) {
	case "geoip":
		c := geoIPConfig{}

		err = unmarshalConfig(cfg, &c)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal GeoIP checker config: %w", err)
		}

		gi, err := newGeoIPChecker(c)
		if err != nil {
			return nil, fmt.Errorf("cannot create GeoIP checker: %w", err)
		}

		return &checkerWithKind{checker: gi, kind: "geoip"}, nil

	case "list":
		c := listCheckerConfig{}

		err = unmarshalConfig(cfg, &c)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal list checker config: %w", err)
		}

		list, err := newListChecker(c)
		if err != nil {
			return nil, fmt.Errorf("cannot create list checker: %w", err)
		}

		return &checkerWithKind{checker: list, kind: "list"}, nil

	case "field":
		c := fieldCheckerConfig{}

		err = unmarshalConfig(cfg, &c)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal field checker config: %w", err)
		}

		field, err := newFieldChecker(c)
		if err != nil {
			return nil, fmt.Errorf("cannot create field checker: %w", err)
		}

		return &checkerWithKind{checker: field, kind: "field"}, nil

	case "reverse_dns":
		c := reverseDNSCheckerConfig{}

		err = unmarshalConfig(cfg, &c)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal reverse DNS checker config: %w", err)
		}

		rdns, err := newReverseDNSChecker(c)
		if err != nil {
			return nil, fmt.Errorf("cannot create reverse DNS checker: %w", err)
		}

		return &checkerWithKind{checker: rdns, kind: "reverse_dns"}, nil

	default:
		return nil, fmt.Errorf("unknown checker %q", kindOnly.Kind)
	}
}

func unmarshalConfig(basic, specified interface{}) error {

	b, err := yaml.Marshal(basic)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, specified)
}
