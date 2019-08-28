package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/time"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)
	local, dir := db.LoadTempDB(assert)
	fmt.Println(dir)
	api := New(local)

	defer time.SetFake()()

	const invalidRequest = ""
	const invalidRequestError = "unexpected end of JSON input"
	const code = `function add(key, d) { var v = db.get(key) || 0; v += d; db.put(key, v); return v; }`

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
		{"put", `{"key": "foo"}`, ``, "data field is required"},
		{"put", `{"key": "foo", "data": null}`, ``, "data field is required"},
		{"put", `{"key": "foo", "data": "bar"}`, `{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, ""},
		{"getRoot", `{}`, `{"root":"3aktuu35stgss7djb5famn6u7iul32nv"}`, ""}, // getRoot when db did change

		// has
		{"has", invalidRequest, ``, invalidRequestError},
		{"has", `{"key": "foo"}`, `{"has":true}`, ""},

		// get
		{"get", invalidRequest, ``, invalidRequestError},
		{"get", `{"key": "foo"}`, `{"has":true,"data":"bar"}`, ""},

		// putBundle
		{"putBundle", invalidRequest, ``, invalidRequestError},
		{"putBundle", fmt.Sprintf(`{"code": "%s"}`, code), `{"root":"mrbevq1sg25j8t86f60oq88nis40ud01"}`, ""},

		// getBundle
		{"getBundle", invalidRequest, ``, invalidRequestError},
		{"getBundle", `{}`, fmt.Sprintf(`{"code":"%s"}`, code), ""},

		// exec
		{"exec", invalidRequest, ``, invalidRequestError},
		{"exec", `{"name": "add", "args": ["bar", 2]}`, `{"result":2,"root":"lchcgvko3ou4ar43lhs23r30os01o850"}`, ""},
		{"get", `{"key": "bar"}`, `{"has":true,"data":2}`, ""},

		// sync
		{"sync", invalidRequest, ``, invalidRequestError},
		{"sync", fmt.Sprintf(`{"remote":"%s"}`, remoteDir), `{"root":"lchcgvko3ou4ar43lhs23r30os01o850"}`, ""},
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
