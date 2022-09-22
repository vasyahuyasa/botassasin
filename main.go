package main

import (
	"context"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vasyahuyasa/botassasin/log"

	_ "net/http/pprof"
)

const configFile = "config.yml"
const defaultMetricsAddr = "0.0.0.0:2112"

var (
	checkerSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "botassasin_checker_duration_seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"checker"})

	totalLinesCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "botassasin_records_processed_total",
	}, []string{"kind"})

	blockSummary = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "botassasin_block_action_duration_seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
)

func main() {
	cfg := readConfig()

	log.EnableDebug(cfg.Debug)

	parser, err := newLogParser(cfg.LogFormat)
	if err != nil {
		log.Fatalf("cannot create log parser: %v", err)
	}

	log.Println("log format:", cfg.LogFormat)

	metricsAddr := defaultMetricsAddr
	if cfg.MetricsAddr != "" {
		metricsAddr = cfg.MetricsAddr
	}

	go func() {
		err := setUpMetricServer(metricsAddr)
		if err != nil {
			log.Fatalf("cannot create metric server: %v", err)
		}
	}()

	measureFn := func(name string, seconds float64) {
		checkerSummary.WithLabelValues(name).Observe(float64(seconds))
	}

	cn, err := newChainFromConfig(cfg, measureFn)
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

	lp, err := newLogPrinter(cfg.Blocklog, cfg.BlocklogTemplate)
	if err != nil {
		log.Fatalf("cannot create log printer: %v", err)
	}

	hitCounter := func(name string) {
		totalLinesCounter.WithLabelValues(name).Inc()
	}

	timeMeasurer := func(seconds float64) {
		blockSummary.Observe(seconds)
	}

	app := newAppCore(logStream, cn, act, lp, cfg.WhitelistCachePath, hitCounter, timeMeasurer)

	log.Printf("watch %s", cfg.Logfile)

	err = app.run()

	if err != nil {
		log.Printf("log streamer exit with error: %v", err)
	}
}

func setUpMetricServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}

func readConfig() config {
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("cannot open config file %s: %v", configFile, err)
	}

	defer f.Close()

	cfg, err := loadConfig(f)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	return cfg
}
