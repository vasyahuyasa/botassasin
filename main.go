package main

import (
	"context"
	"fmt"
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

	logStream, err := newLogStreamer(context.Background(), streamfile)
	if err != nil {
		log.Fatalf("cannot initialize log stre: %v", err)
	}

	log.Printf("watch %s", cfg.Logfile)

	for ln := range logStream.C() {
		fmt.Println(ln)
		if cn.NeedBan(ln) {

		}
	}

	if streamErr := logStream.Err(); streamErr != nil {
		log.Default().Printf("log streamer done with error: %v", err)
	}
}
