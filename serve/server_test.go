package serve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	"roci.dev/diff-server/util/time"
)

func TestAPI(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	tc := []struct {
		rpc         string
		pullReq     string
		expCVReq    *servetypes.ClientViewRequest
		CVResponse  servetypes.ClientViewResponse
		CVErr       error
		expPullResp string
		expPullErr  string
	}{
		// No client view to fetch from.
		{"handlePullRequest",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			nil,
			servetypes.ClientViewResponse{},
			nil,
			`{"stateID":"1pgvpub8mgd4jlsu17qmd3ro0gr3u6hp","patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"c4e7090d"}`,
			""},

		// Successful client view fetch.
		{"handlePullRequest",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			&servetypes.ClientViewRequest{ClientID: "clientid"},
			servetypes.ClientViewResponse{ClientView: []byte(`{"new": "value"}`), LastTransactionID: "1"},
			nil,
			`{"stateID":"mppk37oivnolpnqso4p0hshgas8fc6er","patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"f9ef007b"}`,
			""},

		// Fetch errors out.
		{"handlePullRequest",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			&servetypes.ClientViewRequest{ClientID: "clientid"},
			servetypes.ClientViewResponse{ClientView: []byte(`{"new": "value"}`), LastTransactionID: "1"},
			errors.New("boom"),
			// TODO fritz when we return last transaction id in pull response we need to ensure it is 0 here and not 1
			`{"stateID":"1pgvpub8mgd4jlsu17qmd3ro0gr3u6hp","patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"c4e7090d"}`,
			""},

		// No clientID passed in.
		{"handlePullRequest",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000"}`,
			nil,
			servetypes.ClientViewResponse{},
			nil,
			``,
			"Missing ClientID"},
	}

	for i, t := range tc {
		var cvg clientViewGetter
		var fcvg *fakeClientViewGet
		if t.expCVReq != nil {
			fcvg = &fakeClientViewGet{resp: t.CVResponse, err: t.CVErr}
			cvg = fcvg
		}

		db, s := startTestServer(assert, cvg)
		m := kv.NewMapFromNoms(db.Noms(), types.NewMap(db.Noms(), types.String("foo"), types.String("bar")))
		err := db.PutData(m.NomsMap(), types.String(m.Checksum().String()))
		assert.NoError(err)

		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.pullReq)
		resp, err := http.Post(fmt.Sprintf("http://localhost:8674/%s", t.rpc), "application/json", strings.NewReader(t.pullReq))
		assert.NoError(err, msg)
		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Body)
		assert.NoError(err, msg)
		if t.expPullErr == "" {
			assert.Equal("application/json", resp.Header.Get("Content-type"))
			assert.Equal(t.expPullResp+"\n", string(body.Bytes()), msg)
		} else {
			assert.Regexp(t.expPullErr, string(body.Bytes()), msg)
		}
		if t.expCVReq != nil {
			assert.True(fcvg.called)
			assert.Equal(*t.expCVReq, fcvg.gotReq)
		}

		s.Shutdown(context.Background())
	}
}

type fakeClientViewGet struct {
	resp servetypes.ClientViewResponse
	err  error

	called bool
	gotReq servetypes.ClientViewRequest
}

func (f *fakeClientViewGet) Get(req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, error) {
	f.called = true
	f.gotReq = req
	return f.resp, f.err
}

func startTestServer(assert *assert.Assertions, cvg clientViewGetter) (*db.DB, *http.Server) {
	svr := make(chan *http.Server)
	var d *db.DB
	go func() {
		serverDir, err := ioutil.TempDir("", "")
		fmt.Printf("server dir: %s\n", serverDir)
		assert.NoError(err)
		sp, err := spec.ForDatabase(serverDir)
		assert.NoError(err)
		s, err := newServer(sp.NewChunkStore(), "", cvg)
		assert.NoError(err)
		d = s.db
		hs := http.Server{
			Addr:    ":8674",
			Handler: s,
		}
		svr <- &hs
		err = hs.ListenAndServe()
		assert.Equal(http.ErrServerClosed, err)
	}()
	return d, <-svr
}
