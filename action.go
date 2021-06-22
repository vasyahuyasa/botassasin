package main

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
)

type cmdParams struct {
	IP string
}

type action struct {
	tpl *template.Template
}

func newAction(tplCmd string) (*action, error) {
	tpl, err := template.New("cmd").Parse(tplCmd)
	if err != nil {
		return nil, fmt.Errorf("cannot parse command template %v: %w", tpl, err)
	}

	return &action{
		tpl: tpl,
	}, nil
}

func (a *action) Execute(l logLine) error {
	strCmd, err := a.formatCmdTpl(l)
	if err != nil {
		return fmt.Errorf("cannot format command template %v: %w", a.tpl, err)
	}

	log.Printf("run %s", strCmd)

	return nil
}

func (a *action) formatCmdTpl(l logLine) (string, error) {
	var buf bytes.Buffer

	params := cmdParams{
		IP: l.IP().String(),
	}

	err := a.tpl.Execute(&buf, params)

	return buf.String(), err
}
