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

func newLogLine() *logLine {
	return &logLine{
		fields: map[string]string{},
	}
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

func (l *logLine) Set(field, v string) {
	l.fields[field] = v
}

func (l *logLine) EachField(fn func(key, value string)) {
	for k, v := range l.fields {
		fn(k, v)
	}
}

func (p *logParser) Parse(str string) *logLine {
	matches := p.re.FindStringSubmatch(str)

	if p.mapping == nil {
		p.makeMapping()
	}

	l := newLogLine()

	for name, i := range p.mapping {
		if name == "ip" {
			l.ip = net.ParseIP(matches[i])
			continue
		}

		l.Set(name, matches[i])
	}

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
