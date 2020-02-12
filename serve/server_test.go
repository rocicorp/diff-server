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
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/util/time"
)

func TestAPI(t *testing.T) {
	assert := assert.New(t)

	defer startTestServer(assert).Shutdown(context.Background())
	defer time.SetFake()()

	const code = `function add(id, d) { var v = db.get(id) || 0; v += d; db.put(id, v); return v; }`
	tc := []struct {
		rpc              string
		req              string
		expectedResponse string
		expectedError    string
	}{
		// Lifted mostly from api_test.go
		// We don't need to test everything here, just a smoke test that api tests via http are working!
		{"getRoot", `{}`, `{"root":"klra597i7o2u52k222chv2lqeb13v5sd"}`, ""},
		{"put", `{"id": "foo", "value": "bar"}`, `{"root":"luskchgmo38ohffb2vh9tmfel0ibbfpa"}`, ""},
		{"has", `{"id": "foo"}`, `{"has":true}`, ""},
		{"get", `{"id": "foo"}`, `{"has":true,"value":"bar"}`, ""},
		{"putBundle", fmt.Sprintf(`{"code": "%s"}`, code), `{"root":"n40amopvr0atv1bs77fc30np36a5atse"}`, ""},
		{"getBundle", `{}`, fmt.Sprintf(`{"code":"%s"}`, code), ""},
		{"exec", `{"name": "add", "args": ["bar", 2]}`, `{"result":2,"root":"fs116gjgsjhkpjn8i8jtpno1tbhkhl2s"}`, ""},
		{"get", `{"id": "bar"}`, `{"has":true,"value":2}`, ""},
		{"put", `{"id": "foopa", "value": "doopa"}`, `{"root":"qtjr71od13utmars8d0g4d88or63vh71"}`, ""},
		{"handleSync", `{"basis": "fs116gjgsjhkpjn8i8jtpno1tbhkhl2s"}`,
			`{"patch":[{"op":"add","path":"/u/foopa","value":"doopa"}],"commitID":"qtjr71od13utmars8d0g4d88or63vh71","nomsChecksum":"kgrbb68en2h53f797jl1cpdt89a72rri"}`, ""},
		{"scan", `{"prefix": "foo"}`, `[{"id":"foo","value":"bar"},{"id":"foopa","value":"doopa"}]`, ""},
		{"execBatch", `[{"name": "add", "args": ["bar", 2]},{"name": "add", "args": ["bar", 2]}]`, `{"batch":[{"result":4},{"result":6}],"root":"jkp0ojvvrho7gfpiu5m6164m8alsqkmf"}`, ""},
		{"get", `{"id": "bar"}`, `{"has":true,"value":6}`, ""},
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

func startTestServer(assert *assert.Assertions) *http.Server {
	svr := make(chan *http.Server)
	go func() {
		serverDir, err := ioutil.TempDir("", "")
		fmt.Printf("server dir: %s\n", serverDir)
		assert.NoError(err)
		sp, err := spec.ForDatabase(serverDir)
		assert.NoError(err)
		s, err := newServer(sp.NewChunkStore(), "")
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
