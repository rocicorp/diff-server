package loghttp

import (
	"testing"
)

func TestHeaderWhitelist_Filter(t *testing.T) {
	tests := []struct {
		name      string
		whitelist []string
		httpReq   string
		want      string
	}{
		{
			"not http: empty input",
			[]string{},
			"",
			"",
		},
		{
			"not http: random text",
			[]string{},
			"This is not HTTP",
			"This is not HTTP",
		},
		{
			"request: empty whitelist, no headers or body",
			[]string{},
			"GET / HTTP/1.0\r\n\r\n",
			"GET / HTTP/1.0\r\n\r\n",
		},
		{
			"request: empty whitelist, no headers or body",
			[]string{},
			"GET / HTTP/1.0\r\n\r\n",
			"GET / HTTP/1.0\r\n\r\n",
		},
		{
			"request: empty whitelist, headers no body",
			[]string{},
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n",
			"GET / HTTP/1.0\r\n\r\n",
		},
		{
			"request: empty whitelist, headers and body",
			[]string{},
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\nbody",
			"GET / HTTP/1.0\r\n\r\nbody",
		},
		{
			"request: filters no headers",
			[]string{"Bar", "Foo"},
			"GET / HTTP/1.0\r\nFoo: bar\r\nBar: baz\r\n\r\nbody",
			"GET / HTTP/1.0\r\nFoo: bar\r\nBar: baz\r\n\r\nbody",
		},
		{
			"request: filters some headers",
			[]string{"No-Such-Header", "Bar"},
			"GET / HTTP/1.0\r\nFoo: bar\r\nBar: baz\r\nBonk: boof\r\n\r\nbody",
			"GET / HTTP/1.0\r\nBar: baz\r\n\r\nbody",
		},
		{
			"request: filters all headers",
			[]string{"No-Such-Header"},
			"GET / HTTP/1.0\r\nFoo: bar\r\nBar: baz\r\nBonk: boof\r\n\r\nbody",
			"GET / HTTP/1.0\r\n\r\nbody",
		},
		{
			"response: filters some headers",
			[]string{"No-Such-Header", "Bar"},
			"HTTP/1.0 200 OK\r\nFoo: bar\r\nBar: baz\r\nBonk: boof\r\n\r\nbody",
			"HTTP/1.0 200 OK\r\nBar: baz\r\n\r\nbody",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewHeaderWhitelist(tt.whitelist)
			got := string(hw.Filter([]byte(tt.httpReq)))
			if got != tt.want {
				t.Errorf("HeaderWhitelist.Filter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBodyTrimmer_Filter(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		httpReq string
		want    string
	}{
		{
			"not http: empty input",
			10,
			"",
			"",
		},
		{
			"just a body",
			10,
			"012345678901234567890",
			"01234...90",
		},
		{
			"request: no body",
			10,
			"GET / HTTP/1.0\r\n\r\n",
			"GET / HTTP/1.0\r\n\r\n",
		},
		{
			"response: no body",
			10,
			"HTTP/1.0 200 OK\r\n\r\n",
			"HTTP/1.0 200 OK\r\n\r\n",
		},
		{
			"request: body size smaller than max",
			10,
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\nbody",
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\nbody",
		},
		{
			"request: body size equal to max",
			10,
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n0123456789",
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n0123456789",
		},
		{
			"request: body size larger than max",
			10,
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n012345678901234567890",
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n01234...90",
		},
		{
			"request: max too small",
			1,
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n012345",
			"GET / HTTP/1.0\r\nFoo: bar\r\n\r\n012345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBodyTrimmer(tt.max)
			got := string(bt.Filter([]byte(tt.httpReq)))
			if got != tt.want {
				t.Errorf("BodyTrimmer.Filter() = %q, want %q", got, tt.want)
			}
		})
	}
}
