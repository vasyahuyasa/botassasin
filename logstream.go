package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	checkDelay = time.Millisecond * 300
)

type logStreamer struct {
	ctx    context.Context
	f      *os.File
	err    error
	pos    int64
	parser *logParser
}

func newLogStreamer(ctx context.Context, f *os.File, parser *logParser) (*logStreamer, error) {
	r := &logStreamer{
		ctx:    ctx,
		f:      f,
		parser: parser,
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot stat log: %w", err)
	}

	r.pos = stat.Size()

	_, err = f.Seek(r.pos, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("cannot seek to %d: %w", r.pos, err)
	}

	return r, nil
}

func (r *logStreamer) C() <-chan *logLine {
	c := make(chan *logLine)

	go func(logChan chan *logLine) {
		defer func() {
			close(logChan)
		}()

		for {
			scanner := bufio.NewScanner(r.f)
			for scanner.Scan() {
				s := scanner.Text()
				log := r.parser.Parse(s)
				logChan <- log
			}

			if scanErr := scanner.Err(); scanErr != nil {
				r.err = fmt.Errorf("cannot scan line from log buffer: %w", scanErr)
			}

			time.Sleep(checkDelay)
		}
	}(c)

	return c
}

func (r *logStreamer) Err() error {
	return r.err
}
