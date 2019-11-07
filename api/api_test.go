package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/db"
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

	_, remoteDir := db.LoadTempDB(assert)

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

		// sync
		{"sync", invalidRequest, ``, invalidRequestError},
		{"sync", fmt.Sprintf(`{"remote":"%s"}`, remoteDir), `{"root":"7iavk1o833kqplvrtn2rqc406dfvrf6c"}`, ""},

		// scan
		{"put", `{"id": "foopa", "value": "doopa"}`, `{"root":"61hqku8sbqc76cgjjti99fhkjl3nq4r7"}`, ""},
		{"scan", `{"prefix": "foo"}`, `[{"id":"foo","value":"bar"},{"id":"foopa","value":"doopa"}]`, ""},

		// execBatch
		{"execBatch", invalidRequest, ``, invalidRequestError},
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
