package main

import (
	"fmt"
	"log"
	"strings"
)

const (
	fieldCheckerActionWhitelist fieldCheckerAction = iota
	fieldCheckerActionBan
)

var (
	fieldCheckerActionMap = map[string]fieldCheckerAction{
		"whitelist": fieldCheckerActionWhitelist,
		"block":     fieldCheckerActionBan,
	}

	_ checker = &fieldChecker{}
)

type fieldCheckerAction int

type fieldCheckerConfig struct {
	FieldName string   `yaml:"field_name"`
	Contains  []string `yaml:"contains"`
	Action    string   `yaml:"action"`
}

type fieldChecker struct {
	field    string
	contains []string
	action   fieldCheckerAction
}

func newFieldChecker(cfg fieldCheckerConfig) (*fieldChecker, error) {
	action, ok := fieldCheckerActionMap[cfg.Action]
	if !ok {
		return nil, fmt.Errorf("unknow action %q (supported: whitelist, block)", cfg.Action)
	}

	log.Printf("check field %q contains %v action %s", cfg.FieldName, strings.Join(cfg.Contains, ","), cfg.Action)

	return &fieldChecker{
		field:    cfg.FieldName,
		contains: cfg.Contains,
		action:   action,
	}, nil
}

func (fc *fieldChecker) Check(l *logLine) (harm harmScore, descision instantDecision) {
	fieldVal, ok := l.Get(fc.field)
	if !ok {
		return 0, decisionNone
	}

	for _, v := range fc.contains {
		if strings.Contains(fieldVal, v) {
			if fc.action == fieldCheckerActionWhitelist {
				return 0, decisionWhitelist
			}

			return 0, decisionBan
		}
	}

	return 0, decisionNone
}
