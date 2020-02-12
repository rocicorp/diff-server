package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	gtime "time"

	"github.com/attic-labs/noms/go/spec"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/api/shared"
	"roci.dev/replicant/db"
	jsnoms "roci.dev/replicant/util/noms/json"
	"roci.dev/replicant/util/time"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)
	local, dir := db.LoadTempDB(assert)
	fmt.Println(dir)
	api := New(local)

	defer time.SetFake()()

	const invalidRequest = ""
	const invalidRequestError = "unexpected end of JSON input"
	code, err := json.Marshal(`function add(key, d) { var v = db.get(key) || 0; v += d; db.put(key, v); return v; }
	function log(key, val) { var v = db.get(key) || []; v.push(val); db.put(key, v); }`)
	assert.NoError(err)

	tc := []struct {
		rpc              string
		req              string
		expectedResponse string
		expectedError    string
	}{
		// invalid json for all cases
		// valid json + success case for all cases
		// valid json + failure case for all cases
		// attempt to write non-json with put()
		// attempt to read non-json with get()

		// getRoot on empty db
		{"getRoot", `{}`, `{"root":"klra597i7o2u52k222chv2lqeb13v5sd"}`, ""},

		// put
		{"put", invalidRequest, ``, invalidRequestError},
		{"getRoot", `{}`, `{"root":"klra597i7o2u52k222chv2lqeb13v5sd"}`, ""}, // getRoot when db didn't change
		{"put", `{"id": "foo"}`, ``, "value field is required"},
		{"put", `{"id": "foo", "value": null}`, ``, "value field is required"},
		{"put", `{"id": "foo", "value": "bar"}`, `{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, ""},
		{"getRoot", `{}`, `{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, ""}, // getRoot when db did change

		// has
		{"has", invalidRequest, ``, invalidRequestError},
		{"has", `{"id": "foo"}`, `{"has":true}`, ""},

		// get
		{"get", invalidRequest, ``, invalidRequestError},
		{"get", `{"id": "foo"}`, `{"has":true,"value":"bar"}`, ""},

		// putBundle
		{"putBundle", invalidRequest, ``, invalidRequestError},
		{"putBundle", fmt.Sprintf(`{"code": %s}`, string(code)), `{"root":"vsm77oo1c3r9m5p3r0dkc64imapu1ldm"}`, ""},

		// getBundle
		{"getBundle", invalidRequest, ``, invalidRequestError},
		{"getBundle", `{}`, fmt.Sprintf(`{"code":%s}`, string(code)), ""},

		// exec
		{"exec", invalidRequest, ``, invalidRequestError},
		{"exec", `{"name": "add", "args": ["bar", 2]}`, `{"result":2,"root":"7iavk1o833kqplvrtn2rqc406dfvrf6c"}`, ""},
		{"get", `{"id": "bar"}`, `{"has":true,"value":2}`, ""},

		// handleSync
		{"handleSync", `{"basis":""}`,
			`{"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/u/bar","value":2},{"op":"add","path":"/u/foo","value":"bar"},{"op":"replace","path":"/s/code","value":"function add(key, d) { var v = db.get(key) || 0; v += d; db.put(key, v); return v; }\n\tfunction log(key, val) { var v = db.get(key) || []; v.push(val); db.put(key, v); }"}],"commitID":"7iavk1o833kqplvrtn2rqc406dfvrf6c","nomsChecksum":"2jbp7674jqsv0553qkq0hr68na0tvku1"}`, ""},
		{"handleSync", `{"basis":"bonk"}`, ``, "Invalid basis hash"},
		{"handleSync", `{"basis":"vsm77oo1c3r9m5p3r0dkc64imapu1ldm"}`,
			`{"patch":[{"op":"add","path":"/u/bar","value":2}],"commitID":"7iavk1o833kqplvrtn2rqc406dfvrf6c","nomsChecksum":"2jbp7674jqsv0553qkq0hr68na0tvku1"}`, ""},

		// scan
		{"put", `{"id": "foopa", "value": "doopa"}`, `{"root":"61hqku8sbqc76cgjjti99fhkjl3nq4r7"}`, ""},
		{"scan", `{"prefix": "foo"}`, `[{"id":"foo","value":"bar"},{"id":"foopa","value":"doopa"}]`, ""},
		{"scan", `{"start": {"id": {"value": "foo"}}}`, `[{"id":"foo","value":"bar"},{"id":"foopa","value":"doopa"}]`, ""},
		{"scan", `{"start": {"id": {"value": "foo", "exclusive": true}}}`, `[{"id":"foopa","value":"doopa"}]`, ""},

		// execBatch
		{"execBatch", invalidRequest, ``, invalidRequestError},
		{"execBatch", `[{"name": "add", "args": ["bar", 2]},{"name": ".putBundle", "args": []}]`, `{"error":{"index":1,"detail":"Cannot call system function: .putBundle"},"root":"61hqku8sbqc76cgjjti99fhkjl3nq4r7"}`, ""},
		{"execBatch", `[{"name": "add", "args": ["bar", 2]},{"name": "add", "args": ["bar", 2]},{"name": "log", "args": ["log", "bar"]}]`, `{"batch":[{"result":4},{"result":6},{}],"root":"i3nidc5mep02popavl84u7kt3ged5i14"}`, ""},
		{"get", `{"id": "bar"}`, `{"has":true,"value":6}`, ""},
		// TODO: other scan operators
	}

	for _, t := range tc {
		res, err := api.Dispatch(t.rpc, []byte(t.req))
		if t.expectedError != "" {
			assert.Nil(res, "test case %s: %s", t.rpc, t.req, "test case %s: %s", t.rpc, t.req)
			assert.EqualError(err, t.expectedError, "test case %s: %s", t.rpc, t.req)
		} else {
			assert.Equal(t.expectedResponse, string(res), "test case %s: %s", t.rpc, t.req)
			assert.NoError(err, "test case %s: %s", t.rpc, t.req, "test case %s: %s", t.rpc, t.req)
		}
	}
}

func TestProgress(t *testing.T) {
	twoChunks := [][]byte{[]byte(`"foo`), []byte(`bar"`)}
	assert := assert.New(t)
	db, dir := db.LoadTempDB(assert)
	fmt.Println("dir", dir)
	api := New(db)

	getProgress := func() (received, expected uint64) {
		buf, err := api.Dispatch("syncProgress", mustMarshal(shared.SyncProgressRequest{}))
		assert.NoError(err)
		var resp shared.SyncProgressResponse
		err = json.Unmarshal(buf, &resp)
		assert.NoError(err)
		return resp.BytesReceived, resp.BytesExpected
	}

	totalLength := uint64(len(twoChunks[0]) + len(twoChunks[1]))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-length", fmt.Sprintf("%d", totalLength))
		seen := uint64(0)
		rec, exp := getProgress()
		assert.Equal(uint64(0), rec)
		assert.Equal(uint64(0), exp)
		for _, c := range twoChunks {
			seen += uint64(len(c))
			_, err := w.Write(c)
			assert.NoError(err)
			w.(http.Flusher).Flush()
			gtime.Sleep(100 * gtime.Millisecond)
			rec, exp := getProgress()
			assert.Equal(seen, rec)
			assert.Equal(totalLength, exp)
		}
	}))

	sp, err := spec.ForDatabase(server.URL)
	assert.NoError(err)
	req := shared.SyncRequest{
		Remote:  jsnoms.Spec{sp},
		Shallow: true,
	}

	_, err = api.Dispatch("requestSync", mustMarshal(req))
	assert.Regexp(`Response from [^ ]+ is not valid JSON: json: cannot unmarshal string into Go value of type shared.HandleSyncResponse`, err.Error())
}
