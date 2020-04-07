package serve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

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
		pullMethod  string
		pullReq     string
		authHeader  string
		accountCV   string
		overrideCV  string
		expCVAuth   string
		CVResponse  servetypes.ClientViewResponse
		CVCode      int
		CVErr       error
		expPullResp string
		expPullErr  string
	}{
		// Unsupported method
		{"GET",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			"accountID",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported method: GET"},

		// No client view to fetch from.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			"accountID",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":0,"errorMessage":""}}`,
			""},

		// Successful client view fetch.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful client view fetch via override.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"",
			"override",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful client view fetch via override (with override).
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"override",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does not change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 1},
			200,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 77},
			200,
			nil,
			`{"stateID":"pi99ftvp6nchoej3i58flsqm8enqg4vd","lastMutationID":77,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Fetch errors out.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			errors.New("boom"),
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// No Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing Authorization"},

		// Unknown account passed in Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"BONK",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unknown account"},

		// No clientID passed in.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientViewAuth": "clientauth"}`,
			"accountID",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing clientID"},

		// Invalid baseStateID.
		{"POST",
			`{"baseStateID": "beep", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid baseStateID"},

		// No baseStateID is fine (first pull).
		{"POST",
			`{"baseStateID": "", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"cv",
			"",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Invalid checksum.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "not", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid checksum"},
	}

	for i, t := range tc {
		fcvg := &fakeClientViewGet{resp: t.CVResponse, code: t.CVCode, err: t.CVErr}
		td, _ := ioutil.TempDir("", "")
		s := NewService(td, []Account{Account{ID: "accountID", Name: "accountID", Pubkey: nil, ClientViewURL: t.accountCV}}, t.overrideCV, fcvg, true)
		noms, err := s.getNoms("accountID")
		assert.NoError(err)
		db, err := db.New(noms.GetDataset("client/clientid"))
		assert.NoError(err)
		m := kv.NewMapForTest(noms, "foo", `"bar"`)
		err = db.PutData(m, 1 /*lastMutationID*/)
		assert.NoError(err)

		msg := fmt.Sprintf("test case %d: %s", i, t.pullReq)
		req := httptest.NewRequest(t.pullMethod, "/sync", strings.NewReader(t.pullReq))
		req.Header.Set("Content-type", "application/json")
		if t.authHeader != "" {
			req.Header.Set("Authorization", t.authHeader)
		}
		resp := httptest.NewRecorder()
		s.pull(resp, req)

		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		if t.expPullErr == "" {
			assert.Equal("application/json", resp.Result().Header.Get("Content-type"))
			assert.Equal(t.expPullResp+"\n", string(body.Bytes()), msg)
		} else {
			assert.Regexp(t.expPullErr, string(body.Bytes()), msg)
		}
		expectedCVURL := ""
		if t.overrideCV != "" {
			expectedCVURL = t.overrideCV
		} else if t.accountCV != "" {
			expectedCVURL = t.accountCV
		}
		if expectedCVURL != "" {
			assert.True(fcvg.called, msg)
			assert.Equal(expectedCVURL, fcvg.gotURL, msg)
			assert.Equal(t.expCVAuth, fcvg.gotAuth, msg)
		}
	}
}

type fakeClientViewGet struct {
	resp servetypes.ClientViewResponse
	code int
	err  error

	called  bool
	gotURL  string
	gotAuth string
}

func (f *fakeClientViewGet) Get(url string, req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, int, error) {
	f.called = true
	f.gotURL = url
	f.gotAuth = authToken
	return f.resp, f.code, f.err
}
