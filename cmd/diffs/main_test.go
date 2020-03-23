package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/util/time"
)

func TestServe(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	fmt.Println(dir)

	defer time.SetFake()()

	args := append([]string{"--db=" + dir, "serve", "--port=8674"})
	go impl(args, strings.NewReader(""), os.Stdout, os.Stderr, func(_ int) {})

	const code = `function add(id, d) { var v = db.get(id) || 0; v += d; db.put(id, v); return v; }`
	tc := []struct {
		rpc              string
		req              string
		expectedResponse string
		expectedError    string
	}{
		{"pull", `{"accountID": "sandbox", "baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			`{"stateID":"r0d74qu25vi4dr8fmf58oike0cj4jpth","lastMutationID":0,"patch":[{"op":"remove","path":"/"}],"checksum":"00000000"}`, ""},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.req)
		resp, err := http.Post(fmt.Sprintf("http://localhost:8674/%s", t.rpc), "application/json", strings.NewReader(t.req))
		assert.NoError(err, msg)
		assert.Equal("application/json", resp.Header.Get("Content-type"))
		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Body)
		assert.NoError(err, msg)
		assert.Equal(t.expectedResponse+"\n", string(body.Bytes()), msg)
	}
}

func TestEmptyInput(t *testing.T) {
	assert := assert.New(t)
	db.LoadTempDB(assert)
	var args []string

	// Just testing that they don't crash.
	// See https://github.com/aboodman/replicant/issues/120
	impl(args, strings.NewReader(""), ioutil.Discard, ioutil.Discard, func(_ int) {})
	args = []string{"--db=/tmp/foo"}
	impl(args, strings.NewReader(""), ioutil.Discard, ioutil.Discard, func(_ int) {})
}
