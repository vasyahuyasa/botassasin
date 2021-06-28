package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func Test_listChecker_Check_File(t *testing.T) {
	clear, makeFile := tmpFileCreator()
	defer clear()

	tests := []struct {
		name          string
		config        listCheckerConfig
		logLine       logLine
		wantScore     harmScore
		wantDescision instantDecision
	}{
		{
			name: "simple text file ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeFile(t, `123.123.123.123`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text file ban with comment",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeFile(t, `####
						123.123.123.123 # simple ip addr
						###
						# no need line`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text file blacklist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeFile(t, `123.123.123.123`),
						Type:   "txt",
						Action: "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "simple text file missed",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeFile(t, `123.123.123.122
						123.123.123.124`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionNone,
		},
		{
			name: "simple text mask32 ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeFile(t, `123.123.123.123/32`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text mask16 ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeFile(t, `123.123.0.0/16`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text mask16 whitelist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeFile(t, `123.123.0.0/16`),
						Type:   "txt",
						Action: "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "AWS healthcheck whitelist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeFile(t, `{
							"syncToken": "1624792453",
							"createDate": "2021-06-27-11-14-13",
							"prefixes": [
								{
									"ip_prefix": "3.5.140.0/22",
									"region": "ap-northeast-2",
									"service": "AMAZON",
									"network_border_group": "ap-northeast-2"
								},
								{
									"ip_prefix": "15.177.0.0/18",
									"region": "GLOBAL",
									"service": "ROUTE53_HEALTHCHECKS",
									"network_border_group": "GLOBAL"
								}
							],
							"ipv6_prefixes": []
						}`),
						Type:             "aws_ip_ranges",
						AwsServiceFilter: []string{"ROUTE53_HEALTHCHECKS"},
						Action:           "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(15, 177, 8, 245),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "AWS healthcheck block",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeFile(t, `{
							"syncToken": "1624792453",
							"createDate": "2021-06-27-11-14-13",
							"prefixes": [
								{
									"ip_prefix": "3.5.140.0/22",
									"region": "ap-northeast-2",
									"service": "AMAZON",
									"network_border_group": "ap-northeast-2"
								},
								{
									"ip_prefix": "15.177.0.0/18",
									"region": "GLOBAL",
									"service": "ROUTE53_HEALTHCHECKS",
									"network_border_group": "GLOBAL"
								}
							],
							"ipv6_prefixes": []
						}`),
						Type:             "aws_ip_ranges",
						AwsServiceFilter: []string{"ROUTE53_HEALTHCHECKS"},
						Action:           "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(15, 177, 8, 245),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := newListChecker(tt.config)
			if err != nil {
				t.Fatal(err)
			}

			gotScore, gotDescision := c.Check(tt.logLine)
			if gotScore != tt.wantScore {
				t.Errorf("listChecker.Check() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
			if gotDescision != tt.wantDescision {
				t.Errorf("listChecker.Check() gotDescision = %v, want %v", gotDescision, tt.wantDescision)
			}
		})
	}
}

func Test_listChecker_Check_Webserver(t *testing.T) {
	clear, makeWebserver := tmpWebServerCreator()
	defer clear()

	tests := []struct {
		name          string
		config        listCheckerConfig
		logLine       logLine
		wantScore     harmScore
		wantDescision instantDecision
	}{
		{
			name: "simple text file ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeWebserver(t, `123.123.123.123`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text file ban with comment",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeWebserver(t, `####
						123.123.123.123 # simple ip addr
						###
						# no need line`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text file blacklist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeWebserver(t, `123.123.123.123`),
						Type:   "txt",
						Action: "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "simple text file missed",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeWebserver(t, `123.123.123.122
						123.123.123.124`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionNone,
		},
		{
			name: "simple text mask32 ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeWebserver(t, `123.123.123.123/32`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text mask16 ban",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeWebserver(t, `123.123.0.0/16`),
						Type:   "txt",
						Action: "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
		{
			name: "simple text mask16 whitelist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src:    makeWebserver(t, `123.123.0.0/16`),
						Type:   "txt",
						Action: "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(123, 123, 123, 123),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "AWS healthcheck whitelist",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeWebserver(t, `{
							"syncToken": "1624792453",
							"createDate": "2021-06-27-11-14-13",
							"prefixes": [
								{
									"ip_prefix": "3.5.140.0/22",
									"region": "ap-northeast-2",
									"service": "AMAZON",
									"network_border_group": "ap-northeast-2"
								},
								{
									"ip_prefix": "15.177.0.0/18",
									"region": "GLOBAL",
									"service": "ROUTE53_HEALTHCHECKS",
									"network_border_group": "GLOBAL"
								}
							],
							"ipv6_prefixes": []
						}`),
						Type:             "aws_ip_ranges",
						AwsServiceFilter: []string{"ROUTE53_HEALTHCHECKS"},
						Action:           "whitelist",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(15, 177, 8, 245),
			},
			wantScore:     0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "AWS healthcheck block",
			config: listCheckerConfig{
				Sources: []listCheckerSrcConfig{
					{
						Src: makeWebserver(t, `{
							"syncToken": "1624792453",
							"createDate": "2021-06-27-11-14-13",
							"prefixes": [
								{
									"ip_prefix": "3.5.140.0/22",
									"region": "ap-northeast-2",
									"service": "AMAZON",
									"network_border_group": "ap-northeast-2"
								},
								{
									"ip_prefix": "15.177.0.0/18",
									"region": "GLOBAL",
									"service": "ROUTE53_HEALTHCHECKS",
									"network_border_group": "GLOBAL"
								}
							],
							"ipv6_prefixes": []
						}`),
						Type:             "aws_ip_ranges",
						AwsServiceFilter: []string{"ROUTE53_HEALTHCHECKS"},
						Action:           "block",
					},
				},
			},
			logLine: logLine{
				ip: net.IPv4(15, 177, 8, 245),
			},
			wantScore:     0,
			wantDescision: decisionBan,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := newListChecker(tt.config)
			if err != nil {
				t.Fatal(err)
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "Hello, client")
			}))
			defer ts.Close()

			gotScore, gotDescision := c.Check(tt.logLine)
			if gotScore != tt.wantScore {
				t.Errorf("listChecker.Check() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
			if gotDescision != tt.wantDescision {
				t.Errorf("listChecker.Check() gotDescision = %v, want %v", gotDescision, tt.wantDescision)
			}
		})
	}
}

func tmpWebServerCreator() (clear func(), factory func(t *testing.T, content string) string) {
	var servers []*httptest.Server

	return func() {
			for _, server := range servers {
				server.Close()
			}
		},
		func(t *testing.T, content string) string {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, content)
			}))

			servers = append(servers, server)

			return server.URL
		}
}

func tmpFileCreator() (clear func(), factory func(t *testing.T, content string) string) {
	var files []string

	return func() {
			for _, fname := range files {
				os.Remove(fname)
			}
		},
		func(t *testing.T, content string) string {
			file, err := ioutil.TempFile("", "*.txt")
			if err != nil {
				t.Fatal(err)
			}

			fname := file.Name()
			files = append(files, fname)

			if _, err = file.WriteString(content); err != nil {
				t.Fatal(err)
			}

			defer file.Close()

			return fname
		}
}
