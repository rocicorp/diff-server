package serve

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/time"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)

	defer startTestServer(assert).Shutdown(context.Background())

	d, dir := db.LoadTempDB(assert)
	fmt.Println("client test db", dir)
	assert.NoError(d.PutBundle(types.NewBlob(d.Noms(), strings.NewReader(`function push(key, val) { list = db.get(key) || []; list.push(val); db.put(key, list); }`))))
	_, err := d.Exec("push", types.NewList(d.Noms(), types.String("items"), types.String("foo")))
	assert.NoError(err)
	_, err = d.Exec("push", types.NewList(d.Noms(), types.String("items"), types.String("bar")))
	assert.NoError(err)

	sp, err := spec.ForDatabase("http://localhost:8674")
	assert.NoError(err)
	err = d.Sync(sp)
	assert.NoError(err)

	remote, err := db.New(sp.GetDatabase(), "")
	remote.Reload()
	assert.NoError(err)
	assert.Equal(d.Hash(), remote.Hash())
}
func TestAPI(t *testing.T) {
	assert := assert.New(t)

	defer startTestServer(assert).Shutdown(context.Background())
	defer time.SetFake()()

	const code = `function add(key, d) { var v = db.get(key) || 0; v += d; db.put(key, v); return v; }`
	tc := []struct {
		rpc              string
		req              string
		expectedResponse string
		expectedError    string
	}{
		// Lifted mostly from api_test.go
		// We don't need to test everything here, just a smoke test that api tests via http are working!
		// These hashes should line up with those in api_test.go.
		{"getRoot", `{}`, `{"root":"klra597i7o2u52k222chv2lqeb13v5sd"}`, ""},
		{"put", `{"key": "foo", "data": "bar"}`, `{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, ""},
		{"has", `{"key": "foo"}`, `{"has":true}`, ""},
		{"get", `{"key": "foo"}`, `{"has":true,"data":"bar"}`, ""},
		{"putBundle", fmt.Sprintf(`{"code": "%s"}`, code), `{"root":"mrbevq1sg25j8t86f60oq88nis40ud01"}`, ""},
		{"getBundle", `{}`, fmt.Sprintf(`{"code":"%s"}`, code), ""},
		{"exec", `{"name": "add", "args": ["bar", 2]}`, `{"result":2,"root":"lchcgvko3ou4ar43lhs23r30os01o850"}`, ""},
		{"get", `{"key": "bar"}`, `{"has":true,"data":2}`, ""},
		{"put", `{"key": "foopa", "data": "doopa"}`, `{"root":"v075m8grpbm72rk31gbacf9one3q35ql"}`, ""},
		{"scan", `{"prefix": "foo"}`, `[{"id":"foo","value":"bar"},{"id":"foopa","value":"doopa"}]`, ""},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.req)
		resp, err := http.Post(fmt.Sprintf("http://localhost:8674/%s", t.rpc), "application/json", strings.NewReader(t.req))
		assert.NoError(err, msg)
		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Body)
		assert.NoError(err, msg)
		assert.Equal(t.expectedResponse+"\n", string(body.Bytes()), msg)
	}
}

func startTestServer(assert *assert.Assertions) *http.Server {
	svr := make(chan *http.Server)
	go func() {
		serverDir, err := ioutil.TempDir("", "")
		fmt.Printf("server dir: %s\n", serverDir)
		assert.NoError(err)
		sp, err := spec.ForDatabase(serverDir)
		assert.NoError(err)
		// use same origin used in api_test.go so that hashes match up and any differences with it stand out
		s, err := NewServer(sp.NewChunkStore(), "", "test")
		assert.NoError(err)
		hs := http.Server{
			Addr:    ":8674",
			Handler: s,
		}
		svr <- &hs
		err = hs.ListenAndServe()
		assert.Equal(http.ErrServerClosed, err)
	}()
	return <-svr
}
