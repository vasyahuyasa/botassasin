package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	listCheckerActionWhitelist listCheckerAction = iota
	listCheckerActionBlock

	listCheckerSrcTypeTxt         = "txt"
	listCheckerSrcTypeAWSIpRanges = "aws_ip_ranges"

	httpRequestTimeout = time.Second * 5
)

var listCheckerActionMap = map[string]listCheckerAction{
	"whitelist": listCheckerActionWhitelist,
	"block":     listCheckerActionBlock,
}

var _ checker = &listChecker{}

type listCheckerAction int

type listCheckerSrcConfig struct {
	Src              string   `yaml:"src"`
	Type             string   `yaml:"type"`
	Action           string   `yaml:"action"`
	AwsServiceFilter []string `yaml:"aws_service_filter"`
}

type listCheckerConfig struct {
	Sources []listCheckerSrcConfig
}

type ipList struct {
	ips    []*net.IPNet
	action listCheckerAction
}

type listChecker struct {
	lists []ipList
}

func newListChecker(cfg listCheckerConfig) (*listChecker, error) {
	var lists []ipList

	for _, srcCfg := range cfg.Sources {
		action, ok := listCheckerActionMap[srcCfg.Action]
		if !ok {
			return nil, fmt.Errorf("unknow action %q (supported: whitelist, block)", srcCfg.Action)
		}

		data, err := bytesFromSrc(srcCfg.Src)
		if err != nil {
			return nil, fmt.Errorf("cannot get %s list %q: %w", srcCfg.Type, srcCfg.Src, err)
		}

		switch srcCfg.Type {
		case listCheckerSrcTypeTxt:
			ips := parseTxt(data)
			lists = append(lists, ipList{
				ips:    ips,
				action: action,
			})
			log.Printf("list %s created with %d rules action = %s", srcCfg.Type, len(ips), srcCfg.Action)

		case listCheckerSrcTypeAWSIpRanges:
			ips, err := parseAWSIpRanges(data, srcCfg.AwsServiceFilter)
			if err != nil {
				return nil, fmt.Errorf("cannot parse aws_ip_ranges: %w", err)
			}
			lists = append(lists, ipList{
				ips:    ips,
				action: action,
			})
			log.Printf("list %s created with %d rules action = %s filter = %v", srcCfg.Type, len(ips), srcCfg.Action, srcCfg.AwsServiceFilter)

		default:
			return nil, fmt.Errorf("unknown source type %q (supported types %v)", srcCfg.Type, []string{listCheckerSrcTypeTxt, listCheckerSrcTypeAWSIpRanges})
		}

	}

	return &listChecker{
		lists: lists,
	}, nil
}

func (c *listChecker) Check(l logLine) (score harmScore, descision instantDecision) {
	for _, list := range c.lists {
		if list.contains(l.IP()) {
			if list.action == listCheckerActionWhitelist {
				return 0, decisionWhitelist
			}

			return 0, decisionBan
		}
	}

	return 0, decisionNone
}

func (list *ipList) contains(ip net.IP) bool {
	for _, ipnet := range list.ips {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

func bytesFromSrc(src string) ([]byte, error) {
	// TODO: file cache

	// read file over network
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := http.Client{
			Timeout: httpRequestTimeout,
		}
		res, err := client.Get(src)
		if err != nil {
			return nil, fmt.Errorf("cannot perform GET query to %q: %w", src, err)
		}

		defer res.Body.Close()

		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read from %q: %w", src, err)
		}

		return data, nil
	}

	// read local file
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("cannot read from %q: %w", src, err)
	}

	return data, nil
}

func parseTxt(data []byte) []*net.IPNet {
	scanner := bufio.NewScanner(bytes.NewBuffer(data))

	var ips []*net.IPNet

	for scanner.Scan() {
		// remove comments and trim
		str := strings.TrimSpace(strings.Split(scanner.Text(), "#")[0])

		ipnet, err := parseIPorCIDR(str)
		if err != nil {
			log.Printf("cannot parse %q: %v", str, err)
			continue
		}

		ips = append(ips, ipnet)
	}

	return ips
}

func parseAWSIpRanges(data []byte, filter []string) ([]*net.IPNet, error) {
	// some field are ommited
	type awsIpRanges struct {
		Prefixes []struct {
			IpPrefix string `json:"ip_prefix"`
			Service  string `json:"service"`
		} `json:"prefixes"`
	}

	var ranges awsIpRanges

	err := json.Unmarshal(data, &ranges)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal aws ip range data: %w", err)
	}

	var ips []*net.IPNet

	for _, r := range ranges.Prefixes {
		if strInSlice(r.Service, filter) {
			ipnet, err := parseIPorCIDR(r.IpPrefix)
			if err != nil {
				log.Printf("cannot parse %q: %v", r.IpPrefix, err)
				continue
			}

			ips = append(ips, ipnet)
		}
	}

	return ips, nil
}

func strInSlice(str string, all []string) bool {
	for _, v := range all {
		if v == str {
			return true
		}
	}

	return false
}
