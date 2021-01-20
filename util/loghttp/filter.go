package loghttp

import (
	"bytes"
	"fmt"
	"regexp"
)

// FilterFunc filters an HTTP request or response dump, eg to remove or redact
// header lines.
type FilterFunc func([]byte) []byte

// HeaderAllowlist is a FilterFunc that removes all headers not on
// its allowlist. An empty allowlist filters all headers.
type HeaderAllowlist struct {
	re *regexp.Regexp
}

// NewHeaderAllowlist returns a new HeaderAllowlist that filters
// all but the headers listed in allowlist.
func NewHeaderAllowlist(allolist []string) HeaderAllowlist {
	// Build a regexp that will match header lines on the allowlist.
	// Default to matching none.
	reStr := "^$"
	if len(allolist) > 0 {
		reStr = fmt.Sprintf(`^(%s`, allolist[0])
		for i := 1; i < len(allolist); i++ {
			reStr = fmt.Sprintf("%s|%s", reStr, allolist[i])
		}
		reStr = fmt.Sprintf("%s):", reStr)
	}
	return HeaderAllowlist{regexp.MustCompile(reStr)}
}

// Filter filters the given HTTP request/response dump.
func (hw HeaderAllowlist) Filter(httpReq []byte) []byte {
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

// BodyElider is a FilterFunc that limits the size of the HTTP
// request/response body to max bytes. Bytes are clipped from the
// middle of the body, replaced with "...".
type BodyElider struct {
	max int
}

// NewBodyElider returns a new BodyElider. The minimum value for
// max is 6 to avoid annoying checks on negative indexes when eliding.
func NewBodyElider(max int) BodyElider {
	if max < 6 {
		max = 6
	}
	return BodyElider{max}
}

// Filter filters an HTTP request or response body to max size.
func (be BodyElider) Filter(httpReq []byte) []byte {
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
	if len(body) <= be.max {
		return httpReq
	}
	size := beginBodyIndex + be.max
	filtered := make([]byte, size)
	l := copy(filtered, httpReq[:beginBodyIndex])
	l += copy(filtered[l:], body[:be.max/2])
	l += copy(filtered[l:], []byte("..."))
	l += copy(filtered[l:], body[len(body)-(size-l):])
	return filtered[:l]
}
