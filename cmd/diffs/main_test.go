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

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/util/time"
)

func TestDrop(t *testing.T) {
	assert := assert.New(t)
	tc := []struct {
		in      string
		errs    string
		deleted bool
	}{
		{"no\n", "", false},
		{"N\n", "", false},
		{"balls\n", "", false},
		{"n\n", "", false},
		{"windows\r\n", "", false},
		{"y\n", "", true},
		{"y\r\n", "", true},
	}

	for i, t := range tc {
		d, dir := db.LoadTempDB(assert)
		err := d.PutData(types.NewMap(d.Noms(), types.String("foo"), types.String("bar")))
		assert.NoError(err)

		desc := fmt.Sprintf("test case %d, input: %s", i, t.in)
		args := append([]string{"--db=" + dir, "drop"})
		out := strings.Builder{}
		errs := strings.Builder{}
		code := 0
		impl(args, strings.NewReader(t.in), &out, &errs, func(c int) { code = c })

		assert.Equal(dropWarning, out.String(), desc)
		assert.Equal(t.errs, errs.String(), desc)
		assert.Equal(0, code, desc)
		sp, err := spec.ForDatabase(dir)
		assert.NoError(err)
		noms := sp.GetDatabase()
		ds := noms.GetDataset(db.LOCAL_DATASET)
		assert.Equal(!t.deleted, ds.HasHead())
	}
}

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
		{"handleSync", `{"basis": "00000000000000000000000000000000"}`,
			`{"commitID":"rub50suqvv4fuv48andafkg65i3c184b","patch":[{"op":"remove","path":"/"}],"nomsChecksum":"t13tdcmq2d3pkpt9avk4p4nbt1oagaa3"}`, ""},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.req)
		resp, err := http.Post(fmt.Sprintf("http://localhost:8674/sandbox/foo/%s", t.rpc), "application/json", strings.NewReader(t.req))
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