package main

import (
	"net"
	"reflect"
	"testing"
)

func Test_parseIPorCIDR(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		args    args
		want    *net.IPNet
		wantErr bool
	}{
		{
			name: "ip",
			args: args{
				ip: "123.123.123.123",
			},
			want: &net.IPNet{
				IP:   net.ParseIP("123.123.123.123"),
				Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0xff),
			},
		},
		{
			name: "cidr 32",
			args: args{
				ip: "123.123.123.123/32",
			},
			want: &net.IPNet{
				IP:   net.IP{123, 123, 123, 123},
				Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0xff),
			},
		},
		{
			name: "cidr 24",
			args: args{
				ip: "123.123.123.123/24",
			},
			want: &net.IPNet{
				IP:   net.IP{123, 123, 123, 0},
				Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0x0),
			},
		},
		{
			name: "cidr 28",
			args: args{
				ip: "123.123.123.123/28",
			},
			want: &net.IPNet{
				IP:   net.IP{123, 123, 123, 112},
				Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0xf0),
			},
		},
		{
			name: "cidr 8",
			args: args{
				ip: "123.123.123.123/8",
			},
			want: &net.IPNet{
				IP:   net.IP{123, 0, 0, 0},
				Mask: net.IPv4Mask(0xff, 0x0, 0x0, 0x0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPorCIDR(tt.args.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPorCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIPorCIDR() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
