package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/db"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)
	db, dir := db.LoadTempDB(assert)
	fmt.Println(dir)
	api := New(db)

	const invalidRequest = ""
	const invalidRequestError = "unexpected end of JSON input"
	const code = `function add(key, d) { var v = db.get(key) || 0; db.put(key, v + d); }`

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

		// put
		{"put", invalidRequest, ``, invalidRequestError},
		{"put", `{"key": "foo"}`, ``, "data field is required"},
		{"put", `{"key": "foo", "data": null}`, ``, "data field is required"},
		{"put", `{"key": "foo", "data": "bar"}`, `{}`, ""},

		// has
		{"has", invalidRequest, ``, invalidRequestError},
		{"has", `{"key": "foo"}`, `{"has":true}`, ""},

		// get
		{"get", invalidRequest, ``, invalidRequestError},
		{"get", `{"key": "foo"}`, `{"has":true,"data":"bar"}`, ""},

		// putBundle
		{"putBundle", invalidRequest, ``, invalidRequestError},
		{"putBundle", fmt.Sprintf(`{"code": "%s"}`, code), `{}`, ""},

		// getBundle
		{"getBundle", invalidRequest, ``, invalidRequestError},
		{"getBundle", `{}`, fmt.Sprintf(`{"code":"%s"}`, code), ""},

		// exec
		{"exec", invalidRequest, ``, invalidRequestError},
		{"exec", `{"name": "add", "args": ["bar", 2]}`, `{}`, ""},
		{"get", `{"key": "bar"}`, `{"has":true,"data":2}`, ""},
	}

	for _, t := range tc {
		res, err := api.Dispatch(t.rpc, []byte(t.req))
		if t.expectedError != "" {
			assert.Nil(res, "test case %s: %s", t.rpc, t.req)
			assert.EqualError(err, t.expectedError, "test case %s: %s", t.rpc, t.req)
		} else {
			assert.Equal([]byte(t.expectedResponse), res, "test case %s: %s", t.rpc, t.req)
			assert.NoError(err, "test case %s: %s", t.rpc, t.req)
		}
	}
}
