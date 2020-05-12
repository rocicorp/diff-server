package loghttp

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"

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
		log.Printf("--> %s %s %s\n", req.Method, req.URL, string(buf.Bytes()))
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
		log.Printf("<-- %d %s %s\n", resp.StatusCode, resp.Request.URL, string(buf.Bytes()))
	}
}
