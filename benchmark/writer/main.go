package main

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultDelayAfterWrite = time.Nanosecond * 1_000_000
	reportDelaySeconds     = 1

	metricsAddr = ":12383"
)

var (
	preparedLogs = [1][]byte{
		[]byte(`176.59.46.121 - - [23/Aug/2022:09:52:09 +0000] "GET /api/application/items/?item_ids=3633 HTTP/2.0" 200 22 "https://pizzafabrika.ru/order.html" "Mozilla/5.0 (Linux; Android 10; HRY-LX1T Build/HONORHRY-LX1T; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/103.0.5060.129 Mobile Safari/537.36" rt=0.066 uct="0.001" uht="0.067" urt="0.067"
`),
	}

	linesWrittenBeforeReport uint64

	linesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "botassasin_writer_lines_written_total",
	})
)

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Usage: logwriter <file> [delay ms]")
	}

	logfile := os.Args[1]

	f, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		log.Fatalf("cannot open %s: %v", logfile, err)
	}
	defer func() {
		f.Close()
	}()

	go func() {
		err := setUpMetricServer()
		if err != nil {
			log.Fatalf("cannot create metric server: %v", err)
		}
	}()

	delayAfterWrite := defaultDelayAfterWrite

	if len(os.Args) >= 3 {
		nano, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("cannot parse as number %q: %v", os.Args[2], err)
		}

		delayAfterWrite = time.Nanosecond * time.Duration(nano)
	}

	err = writer(f, delayAfterWrite)
	if err != nil {
		log.Fatalf("writer failed: %v", err)
	}
}

func writer(w io.Writer, delayAfterWrite time.Duration) error {
	var err error

	go report()

	for {
		_, err = w.Write(preparedLogs[0])
		if err != nil {
			return err
		}

		linesCounter.Inc()
		atomic.AddUint64(&linesWrittenBeforeReport, 1)
		time.Sleep(delayAfterWrite)
	}
}

func report() {
	ticker := time.NewTicker(time.Second * reportDelaySeconds)

	for range ticker.C {
		linesWritten := atomic.SwapUint64(&linesWrittenBeforeReport, 0)
		log.Printf("%d lines/sec", linesWritten/reportDelaySeconds)
	}

}

func setUpMetricServer() error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(metricsAddr, nil)
}
