package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"text/template"
)

type cmdParams map[string]string

type action struct {
	params []*template.Template
}

func newAction(parmTpls []string) (*action, error) {
	var tpls []*template.Template

	for i, tplCmd := range parmTpls {
		tpl, err := template.New(fmt.Sprintf("param_%d", i)).Parse(tplCmd)
		if err != nil {
			return nil, fmt.Errorf("cannot parse command template %v: %w", tpl, err)
		}

		tpls = append(tpls, tpl)
	}

	return &action{
		params: tpls,
	}, nil
}

func (a *action) Execute(l logLine) error {
	// no action
	if len(a.params) == 0 {
		return nil
	}

	strCmd, cmdParams, err := a.formatCmdTpl(l)
	if err != nil {
		return fmt.Errorf("cannot format command template: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})

	cmd := exec.Command(strCmd, cmdParams...)
	cmd.Stdout = buf
	cmd.Stderr = buf

	err = cmd.Run()

	if buf.Len() != 0 {
		log.Println("action output:", buf.String())
	}

	return err
}

func (a *action) formatCmdTpl(l logLine) (string, []string, error) {
	params := cmdParams{
		"IP": l.IP().String(),
	}

	for _, field := range l.Fields() {
		v, _ := l.Get(field)
		params[field] = v
	}

	var cmd string
	var cmdParams []string

	for i, tpl := range a.params {
		buf := bytes.NewBuffer([]byte{})

		err := tpl.Execute(buf, params)
		if err != nil {
			return "", nil, fmt.Errorf("param %d: %w", err)
		}

		if i == 0 {
			cmd = buf.String()
		} else {
			cmdParams = append(cmdParams, buf.String())
		}
	}

	return cmd, cmdParams, nil
}
