package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const checkDelay = time.Millisecond * 300

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

	go func() {
		for {
			select {
			case <-r.ctx.Done():
				close(c)
				return

			default:
				size, err := r.currentLogSize()
				if err != nil {
					r.err = err
					close(c)
					return
				}

				if size != r.pos {
					bufSize := size - r.pos

					// TODO: buffer pool
					buf := make([]byte, bufSize)

					n, err := r.f.Read(buf)
					if err != nil {
						r.err = fmt.Errorf("cannot read from log: %w", err)
						close(c)
						return
					}

					if int64(n) != bufSize {
						r.err = fmt.Errorf("expected read %d bytes but got %d", bufSize, n)
						close(c)
						return
					}

					scanner := bufio.NewScanner(bytes.NewBuffer(buf))
					for scanner.Scan() {
						s := scanner.Text()
						log := r.parser.Parse(s)
						c <- log
					}

					if scanErr := scanner.Err(); scanErr != nil {
						r.err = fmt.Errorf("cannot scan line from log buffer: %w", scanErr)
						close(c)
						return
					}

					r.pos = size

					continue
				}

				time.Sleep(checkDelay)
			}
		}

	}()

	return c

}

func (r *logStreamer) Err() error {
	return r.err
}

func (r *logStreamer) currentLogSize() (int64, error) {
	stat, err := r.f.Stat()
	if err != nil {
		return 0, fmt.Errorf("cannot stat log: %w", err)
	}

	return stat.Size(), nil
}
