package loghttp

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	lh "github.com/motemen/go-loghttp"

	// Import go-loghttp/global to override default http transport.
	_ "github.com/motemen/go-loghttp/global"
)

func init() {
	lh.DefaultLogRequest = func(req *http.Request) {
		buf := &bytes.Buffer{}
		if req.Body != nil {
			_, err := io.Copy(buf, req.Body)
			if err != nil {
				log.Printf("ERROR: Could not read request body: %v", err)
				return
			}
		}
		req.Body = ioutil.NopCloser(buf)
		log.Printf("Outgoing --> %s %s %s\n", req.Method, req.URL, string(buf.Bytes()))
	}

	lh.DefaultLogResponse = func(resp *http.Response) {
		buf := &bytes.Buffer{}
		if resp.Body != nil {
			_, err := io.Copy(buf, resp.Body)
			if err != nil {
				log.Printf("ERROR: Could not read response body: %v", err)
				return
			}
		}
		resp.Body = ioutil.NopCloser(buf)
		log.Printf("Outgoing <-- %d %s %s\n", resp.StatusCode, resp.Request.URL, string(buf.Bytes()))
	}
}

// Wrap wraps the given handler with a Handler that logs HTTP requests
// and responses.
func Wrap(handler http.Handler) Handler {
	return Handler{wrapped: handler}
}

// Handler is a wrapper for http.Handlers that logs the HTTP request and
// response. It logs full request headers but logging full response headers
// seems like more work (eg
// https://stackoverflow.com/questions/29319783/logging-responses-to-incoming-http-requests-inside-http-handlefunc)
// so we settle for logging the response status code and response body for now.
type Handler struct {
	wrapped http.Handler
}

// ServeHTTP logs the request, calls the underlying handler, and logs the response.
func (l Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	log.Printf("--> Incoming %s\n", dump)

	rl := &responseLogger{ResponseWriter: w, status: 200}
	l.wrapped.ServeHTTP(rl, r)
	body, err := maybeUnzip(rl.responseBody.Bytes())
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return
	}
	log.Printf("<-- Incoming %d %s\n", rl.status, string(body))
}

func maybeUnzip(b []byte) ([]byte, error) {
	if http.DetectContentType(b) != "application/x-gzip" {
		return b, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(zr)
}

type responseLogger struct {
	http.ResponseWriter
	responseBody bytes.Buffer
	status       int
}

func (r *responseLogger) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseLogger) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	if err != nil {
		return n, err
	}
	return r.responseBody.Write(b)
}
