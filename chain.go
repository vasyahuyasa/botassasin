package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	decisionNone instantDecision = iota
	decisionBan
	decisionWhitelist
)

type instantDecision int

type harmScore int

type checker interface {
	Check(logLine) (harm harmScore, descision instantDecision)
}

type chain struct {
	checkers []checker
}

func newChainFromConfig(cfg config) (*chain, error) {
	var checkers []checker

	for _, checkerCfg := range cfg.Checkers {
		c, err := checkerFromConfig(checkerCfg)
		if err != nil {
			return nil, fmt.Errorf("cannot create checker: %w", err)
		}

		checkers = append(checkers, c)
	}

	return &chain{
		checkers: checkers,
	}, nil
}

func (c *chain) NeedBan(l logLine) bool {
	score := harmScore(0)

	for _, chk := range c.checkers {
		s, decision := chk.Check(l)
		if decision == decisionNone {
			score += s
			continue
		}

		return decision == decisionBan
	}

	return score >= 0
}

func checkerFromConfig(cfg checkerConfig) (checker, error) {
	var kindOnly struct {
		Kind string
	}

	err := unmarshalConfig(cfg, &kindOnly)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal checker config: %w", err)
	}

	switch strings.ToLower(kindOnly.Kind) {
	case "whitelist":
		c := whitelistConfig{}

		err = unmarshalConfig(cfg, &c)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal whitelist config: %w", err)
		}

		wl, err := newWhitelistChecker(c)
		if err != nil {
			return nil, fmt.Errorf("cannot create whitelist: %w", err)
		}

		return wl, nil

	default:
		return nil, fmt.Errorf("cannot create whitelist: %w", err)
	}
}

func unmarshalConfig(basic, specified interface{}) error {

	b, err := yaml.Marshal(basic)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, specified)
}
