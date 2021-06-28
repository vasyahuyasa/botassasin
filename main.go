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

	cn, err := newChainFromConfig(cfg)
	if err != nil {
		log.Fatalf("cannot create chain: %v", err)
	}

	streamfile, err := os.Open(cfg.Logfile)
	if err != nil {
		log.Fatalf("cannot open log file %s: %v", cfg.Logfile, err)
	}

	log.Println("format:", cfg.LogFormat)

	parser, err := newLogParser(cfg.LogFormat)
	if err != nil {
		log.Fatalf("cannot create log parser: %v", err)
	}

	logStream, err := newLogStreamer(context.Background(), streamfile, parser)
	if err != nil {
		log.Fatalf("cannot initialize log stre: %v", err)
	}

	act, err := newAction(cfg.BlockAction)
	if err != nil {
		log.Fatalf("cannot create action: %v", err)
	}

	log.Printf("watch %s", cfg.Logfile)

	for ln := range logStream.C() {
		if cn.NeedBan(ln) {
			log.Println("ban", ln.IP())
			if actErr := act.Execute(ln); actErr != nil {
				log.Printf("cannot execute action: %v", err)
			}
		}
	}

	if streamErr := logStream.Err(); streamErr != nil {
		log.Default().Printf("log streamer done with error: %v", err)
	}
}
