package main

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v2"
)

type checkerConfig interface{}

type configBlockAction struct {
	params []string
}

type config struct {
	Logfile     string            `yaml:"logfile"`
	LogFormat   string            `yaml:"log_format"`
	Checkers    []checkerConfig   `yaml:"checkers"`
	BlockAction configBlockAction `yaml:"block_action"`
}

func loadConfig(r io.Reader) (config, error) {
	decoder := yaml.NewDecoder(r)

	cfg := config{}

	err := decoder.Decode(&cfg)
	if err != nil {
		return config{}, fmt.Errorf("cannot decode config: %w", err)
	}

	return cfg, nil
}

func (c *configBlockAction) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	err := unmarshal(&s)
	if err == nil {
		c.params = []string{s}
		return nil
	}

	strs := []string{}

	err = unmarshal(&strs)
	if err != nil {
		return err
	}

	c.params = strs

	return nil
}

func (c configBlockAction) String() string {
	return strings.Join(c.params, " ")
}
