package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/db"
	"roci.dev/replicant/util/time"
)

func TestCommands(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	td, err := ioutil.TempDir("", "")
	fmt.Println("test database:", td)
	assert.NoError(err)

	bf, err := os.Create(path.Join(td, "bundle.js"))
	assert.NoError(err)
	_, err = bf.Write([]byte("function futz(k, v){ db.put(k, v) }; function echo(v) { return v; };\n"))
	assert.NoError(err)
	assert.NoError(bf.Close())

	tc := []struct {
		label string
		in    string
		args  string
		code  int
		out   string
		err   string
	}{
		{
			"log empty",
			"",
			"log --no-pager",
			0,
			"",
			"",
		},
		{
			"exec unknown-function",
			"",
			fmt.Sprintf("exec --bundle=%s monkey", bf.Name()),
			1,
			"",
			"Unknown function: monkey\n",
		},
		{
			"exec missing-key",
			"",
			fmt.Sprintf("exec --bundle=%s futz", bf.Name()),
			1,
			"",
			"Error: Invalid id\n    at bootstrap.js:20:14\n    at bootstrap.js:26:4\n    at futz (bundle.js:1:22)\n    at apply (<native code>)\n    at recv (bootstrap.js:67:12)\n\n",
		},
		{
			"exec missing-val",
			"",
			fmt.Sprintf("exec --bundle=%s futz foo", bf.Name()),
			1,
			"",
			"Error: Invalid value\n    at bootstrap.js:29:15\n    at futz (bundle.js:1:22)\n    at apply (<native code>)\n    at recv (bootstrap.js:67:12)\n\n",
		},
		{
			"exec good",
			"",
			fmt.Sprintf("exec --bundle=%s futz foo bar", bf.Name()),
			0,
			"",
			"",
		},
		{
			"log exec good",
			"",
			"log --no-pager",
			0,
			fmt.Sprintf("commit dq05ge0eu8rh74buuh2tq31pnlsp9bvs\nCreated:     %s\nStatus:      PENDING\nMerged:      %s\nTransaction: futz(\"foo\", \"bar\")\n(root) {\n+   \"foo\": \"bar\"\n  }\n\n", time.Now(), time.Now()),
			"",
		},
		{
			"exec echo",
			"",
			fmt.Sprintf("exec --bundle=%s echo monkey", bf.Name()),
			0,
			`"monkey"`,
			"",
		},
		{
			"has missing-arg",
			"",
			"has",
			1,
			"",
			"required argument 'id' not provided\n",
		},
		{
			"has good",
			"",
			"has foo",
			0,
			"true\n",
			"",
		},
		{
			"get bad missing-arg",
			"",
			"get",
			1,
			"",
			"required argument 'id' not provided\n",
		},
		{
			"get good",
			"",
			"get foo",
			0,
			"\"bar\"\n",
			"",
		},
		{
			"scan all",
			"",
			"scan",
			0,
			"foo: \"bar\"\n",
			"",
		},
		{
			"scan prefix good",
			"",
			"scan --prefix=f",
			0,
			"foo: \"bar\"\n",
			"",
		},
		{
			"scan prefix bad",
			"",
			"scan --prefix=g",
			0,
			"",
			"",
		},
		{
			"scan start-id good",
			"",
			"scan --start-id=foo",
			0,
			"foo: \"bar\"\n",
			"",
		},
		{
			"scan start-id bad",
			"",
			"scan --start-id=g",
			0,
			"",
			"",
		},
		{
			"scan start-id-exclusive good",
			"",
			"scan --start-id=f --start-id-exclusive",
			0,
			"foo: \"bar\"\n",
			"",
		},
		{
			"scan start-id-exclusive bad",
			"",
			"scan --start-id=foo --start-id-exclusive",
			0,
			"",
			"",
		},
		{
			"scan start-index good",
			"",
			"scan --start-index=0",
			0,
			"foo: \"bar\"\n",
			"",
		},
		{
			"scan start-index bad",
			"",
			"scan --start-index=1",
			0,
			"",
			"",
		},
		{
			"del bad missing-arg",
			"",
			"del",
			1,
			"",
			"required argument 'id' not provided\n",
		},
		{
			"del good no-op",
			"",
			"del monkey",
			0,
			"No such id.\n",
			"",
		},
		{
			"del good",
			"",
			"del foo",
			0,
			"",
			"",
		},
	}

	for _, c := range tc {
		ob := &strings.Builder{}
		eb := &strings.Builder{}
		code := 0
		args := append([]string{"--db=" + td}, strings.Split(c.args, " ")...)
		impl(args, strings.NewReader(c.in), ob, eb, func(c int) {
			code = c
		})

		assert.Equal(c.code, code, c.label)
		assert.Equal(c.out, ob.String(), c.label)
		assert.Equal(c.err, eb.String(), c.label)
	}
}

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
		d.Put("foo", types.String("bar"))
		val, err := d.Get("foo")
		assert.NoError(err)
		assert.Equal("bar", string(val.(types.String)))

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
			`{"patch":[{"op":"remove","path":"/"}],"commitID":"uosmsi0mbbd1qgf2m0rgfkcrhf32c7om","nomsChecksum":"t13tdcmq2d3pkpt9avk4p4nbt1oagaa3"}`, ""},
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
