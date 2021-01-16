package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	gt "time"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/account"
	"roci.dev/diff-server/db"
	"roci.dev/diff-server/util/time"
)

func TestServe(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()
	fmt.Println(dir)

	accountDBDir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer func() { assert.NoError(os.RemoveAll(accountDBDir)) }()
	accountDB := account.LoadTempDBWithPath(assert, accountDBDir)
	account.AddUnittestAccount(assert, accountDB)

	cvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"clientView": {}}`))
	}))

	defer time.SetFake()()

	args := append([]string{"--db=" + dir, "--account-db=" + accountDBDir, "serve", "--port=8674"})
	go impl(args, strings.NewReader(""), os.Stdout, os.Stderr, func(_ int) {})

	// Wait for server to start...
	i := 0
	for {
		i++
		gt.Sleep(100 * gt.Millisecond)
		resp, err := http.Get("http://localhost:8674/")
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if i > 20 {
			panic("server never started")
		}
	}

	const code = `function add(id, d) { var v = db.get(id) || 0; v += d; db.put(id, v); return v; }`
	tc := []struct {
		rpc              string
		req              string
		authHeader       string
		expectedResponse string
		expectedError    string
	}{
		{"pull",
			fmt.Sprintf(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewURL": "%s", "version": 3}`, cvServer.URL),
			fmt.Sprintf("%d", account.UnittestID),
			`{"stateID":"r0d74qu25vi4dr8fmf58oike0cj4jpth","lastMutationID":0,"patch":[{"op":"replace","path":"","valueString":"{}"}],"checksum":"00000000","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.req)
		httpReq, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:8674/%s", t.rpc), strings.NewReader(t.req))
		assert.NoError(err)
		httpReq.Header.Add("Authorization", t.authHeader)
		httpReq.Header.Add("Content-type", "application/json")
		resp, err := http.DefaultClient.Do(httpReq)
		assert.NoError(err, msg)
		assert.Equal("application/json", resp.Header.Get("Content-type"))
		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Body)
		assert.NoError(err, msg)
		assert.Equal(t.expectedResponse+"\n", string(body.Bytes()), msg)
	}

	cvServer.Close()
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
