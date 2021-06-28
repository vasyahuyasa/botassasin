package main

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

type checkerConfig interface{}

type config struct {
	Logfile     string          `yaml:"logfile"`
	LogFormat   string          `yaml:"log_format"`
	Checkers    []checkerConfig `yaml:"checkers"`
	BlockAction string          `yaml:"block_action"`
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
