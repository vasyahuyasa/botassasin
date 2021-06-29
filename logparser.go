package main

import (
	"fmt"
	"net"
	"regexp"
)

type logLine struct {
	ip     net.IP
	fields map[string]string
}

type logParser struct {
	re      *regexp.Regexp
	mapping map[string]int
}

func newLogParser(format string) (*logParser, error) {
	re, err := regexp.Compile(format)

	if err != nil {
		return nil, fmt.Errorf("cannot compile %q: %w", format, err)
	}

	return &logParser{re: re}, nil
}

func (l *logLine) IP() net.IP {
	return l.ip
}

func (l *logLine) String() string {
	return fmt.Sprintf("%s %v", l.ip.String(), l.fields)
}

func (l *logLine) Get(field string) (string, bool) {
	str, ok := l.fields[field]
	return str, ok
}

func (p *logParser) Parse(str string) logLine {
	matches := p.re.FindStringSubmatch(str)

	if p.mapping == nil {
		p.makeMapping()
	}

	var l logLine

	fields := map[string]string{}

	for name, i := range p.mapping {
		if name == "ip" {
			l.ip = net.ParseIP(matches[i])
			continue
		}

		fields[name] = matches[i]
	}

	l.fields = fields

	return l
}

func (p *logParser) makeMapping() {
	mapping := map[string]int{}

	for i, name := range p.re.SubexpNames() {
		if i > 0 {
			mapping[name] = i
		}
	}

	p.mapping = mapping
}
