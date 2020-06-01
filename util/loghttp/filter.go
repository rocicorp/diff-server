package loghttp

import (
	"bytes"
	"fmt"
	"regexp"
)

// FilterFunc filters an HTTP request or response dump, eg to remove or redact
// header lines. A FilterFunc must return a valid HTTP dump (otherwise
// they cannot be chained).
type FilterFunc func([]byte) []byte

// HeaderWhitelist is a FilterFunc that removes all headers not on
// its whitelist. An empty whitelist filters all headers.
type HeaderWhitelist struct {
	re *regexp.Regexp
}

// NewHeaderWhitelist returns a new HeaderWhitelist that filters
// all but the headers listed in whitelist.
func NewHeaderWhitelist(whitelist []string) HeaderWhitelist {
	// Build a regexp that will match header lines on the whitelist.
	// Default to matching none.
	reStr := "^$"
	if len(whitelist) > 0 {
		reStr = fmt.Sprintf(`^(%s`, whitelist[0])
		for i := 1; i < len(whitelist); i++ {
			reStr = fmt.Sprintf("%s|%s", reStr, whitelist[i])
		}
		reStr = fmt.Sprintf("%s):", reStr)
	}
	return HeaderWhitelist{regexp.MustCompile(reStr)}
}

// Filter filters the given HTTP request/response dump.
func (hw HeaderWhitelist) Filter(httpReq []byte) []byte {
	// Split the header lines on CRLF.
	endHeadersIndex := bytes.Index(httpReq, []byte("\r\n\r\n"))
	if endHeadersIndex == -1 {
		return httpReq
	}
	headerLines := bytes.Split(httpReq[:endHeadersIndex], []byte("\r\n"))
	if len(headerLines) == 0 {
		return httpReq
	}

	filtered := make([]byte, len(httpReq))

	// Copy the request or status line to output.
	l := copy(filtered, headerLines[0])
	l += copy(filtered[l:], []byte("\r\n"))

	// Filter the header lines.
	for i := 1; i < len(headerLines); i++ {
		if hw.re.Match(headerLines[i]) {
			l += copy(filtered[l:], headerLines[i])
			l += copy(filtered[l:], []byte("\r\n"))
		}
	}
	l += copy(filtered[l:], []byte("\r\n"))

	// If no body we're done. Else copy the body.
	if endHeadersIndex+3 == len(httpReq) {
		return filtered[:l]
	}
	l += copy(filtered[l:], httpReq[endHeadersIndex+4:])

	return filtered[:l]
}

// BodyTrimmer is a FilterFunc that limits the size of the HTTP
// request/response body to max bytes. Bytes are clipped from the
// middle of the body, replaced with "...".
type BodyTrimmer struct {
	max int
}

// NewBodyTrimmer returns a new BodyTrimmer. The minimum value for
// max is 6 to avoid annoying checks on negative indexes when trimming.
func NewBodyTrimmer(max int) BodyTrimmer {
	if max < 6 {
		max = 6
	}
	return BodyTrimmer{max}
}

// Filter filters an HTTP request or response body to max size.
func (bt BodyTrimmer) Filter(httpReq []byte) []byte {
	endHeadersIndex := bytes.Index(httpReq, []byte("\r\n\r\n"))
	beginBodyIndex := 0
	// The dump code doesn't currently dump HTTP response headers.
	// In that case beginBodyIndex is zero. If we do have headers,
	// the body begins after the CRLFCRLF.
	if endHeadersIndex != -1 {
		beginBodyIndex = endHeadersIndex + 4
	}
	// If no body, we're done.
	if beginBodyIndex > len(httpReq) {
		return httpReq
	}
	
	body := httpReq[beginBodyIndex:]
	if len(body) <= bt.max {
		return httpReq
	}
	size := beginBodyIndex + bt.max
	filtered := make([]byte, size)
	l := copy(filtered, httpReq[:beginBodyIndex])
	l += copy(filtered[l:], body[:bt.max/2])
	l += copy(filtered[l:], []byte("..."))
	l += copy(filtered[l:], body[len(body)-(size-l):])
	return filtered[:l]
}
