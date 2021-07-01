package main

import (
	"context"
	"log"
	"os"
)

const configFile = "config.yml"

func main() {
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("cannot open config file %s: %v", configFile, err)
	}

	defer f.Close()

	cfg, err := loadConfig(f)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	parser, err := newLogParser(cfg.LogFormat)
	if err != nil {
		log.Fatalf("cannot create log parser: %v", err)
	}

	log.Println("log format:", cfg.LogFormat)

	cn, err := newChainFromConfig(cfg)
	if err != nil {
		log.Fatalf("cannot create chain: %v", err)
	}

	streamfile, err := os.Open(cfg.Logfile)
	if err != nil {
		log.Fatalf("cannot open log file %s: %v", cfg.Logfile, err)
	}

	logStream, err := newLogStreamer(context.Background(), streamfile, parser)
	if err != nil {
		log.Fatalf("cannot initialize log stre: %v", err)
	}

	act, err := newAction(cfg.BlockAction.params)
	if err != nil {
		log.Fatalf("cannot create action: %v", err)
	}

	log.Printf("block action: %s", cfg.BlockAction)

	log.Printf("watch %s", cfg.Logfile)

	app := newAppCore(logStream, cn, act)

	err = app.run()

	if err != nil {
		log.Printf("log streamer exit with error: %v", err)
	}
}
