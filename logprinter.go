package main

import (
	"fmt"
	"io"
	"os"
	"text/template"
	"time"
)

const timeForamt = "2006-01-02 15:04:05 -0700"

type logPrinterParams map[string]interface{}

type logPrinter struct {
	w io.Writer
	t *template.Template
}

func newLogPrinter(logfile, tpl string) (*logPrinter, error) {
	w := os.Stdout

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}

		w = f
	}

	return newlogPrinterFromWriter(w, tpl)
}

func newlogPrinterFromWriter(w io.Writer, tpl string) (*logPrinter, error) {
	t, err := template.New("log").Parse(tpl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse log template %q: %w", tpl, err)
	}

	return &logPrinter{
		w: w,
		t: t,
	}, nil
}

func (lw *logPrinter) Println(l logLine) error {
	params := logPrinterParams{
		"ip":   l.ip.String(),
		"time": time.Now().Format(timeForamt),
	}

	l.EachField(func(key, value string) {
		params[key] = value
	})

	return lw.write(params)
}

func (lw *logPrinter) write(params logPrinterParams) error {
	err := lw.t.Execute(lw.w, params)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(lw.w, "\n")
	if err != nil {
		return err
	}

	return nil
}
