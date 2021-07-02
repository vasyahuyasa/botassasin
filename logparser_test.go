package main

import (
	"net"
	"reflect"
	"testing"
)

func Test_logParser_Parse(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		re   string
		args args
		want *logLine
	}{
		{
			name: "simple",
			re:   `^(?P<ip>.+) - - \[.*\] \"(?P<method>.+) (?P<request>.+) (?P<proto>.+)\" \d{3} \d+ \"(?P<referer>.*)\" \"(?P<user_agent>.*)\" rt.*$`,
			args: args{
				str: `83.149.21.43 - - [24/Jun/2021:12:02:44 +0000] "GET /api/products/salesitems?district_id=24&channel_id=81 HTTP/2.0" 200 7465 "https://pizzafabrika.ru/order.html" "Mozilla/5.0 (iPhone; CPU iPhone OS 14_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148" rt=0.074 uct="0.000" uht="0.072" urt="0.072"`,
			},
			want: &logLine{
				ip: net.ParseIP("83.149.21.43"),
				fields: map[string]string{
					"method":     "GET",
					"request":    "/api/products/salesitems?district_id=24&channel_id=81",
					"proto":      "HTTP/2.0",
					"referer":    "https://pizzafabrika.ru/order.html",
					"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
				},
			},
		},
		{
			name: "simple2",
			re:   `^(?P<ip>\d+\.\d+\.\d+\.\d+) - - \[.{26}\] \"(?P<request>[^\"]*)\" \d{3} \d+ \"(?P<referer>[^\"]*)\" \"(?P<user_agent>[^\"]*)\" rt.*$`,
			args: args{
				str: `83.149.21.143 - - [24/Jun/2021:12:02:44 +0000] "GET /api/products/salesitems?district_id=24&channel_id=81 HTTP/2.0" 200 7465 "https://pizzafabrika.ru/order.html" "Mozilla/5.0 (iPhone; CPU iPhone OS 14_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148" rt=0.074 uct="0.000" uht="0.072" urt="0.072"`,
			},
			want: &logLine{
				ip: net.ParseIP("83.149.21.143"),
				fields: map[string]string{
					"request":    "GET /api/products/salesitems?district_id=24&channel_id=81 HTTP/2.0",
					"referer":    "https://pizzafabrika.ru/order.html",
					"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := newLogParser(tt.re)
			if err != nil {
				t.Fatalf("cannot create log parser: %v", err)
			}

			if got := p.Parse(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("logParser.Parse() = %#v, \nwant %#v", got, tt.want)
			}
		})
	}
}
